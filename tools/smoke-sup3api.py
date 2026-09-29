"""Check a running Sup3API site without submitting any generation jobs (stdlib only)."""
import argparse
import json
import os
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None  # Never forward a gateway credential to a different destination.


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url', default='http://127.0.0.1:8080')
    parser.add_argument('--assets', action='store_true', help='Check asset authentication, capabilities and free quotes')
    parser.add_argument('--models', action='store_true', help='Query models available to this gateway key')
    args = parser.parse_args()
    base = args.base_url.rstrip('/')
    url = urlsplit(base)
    if url.scheme not in ('http', 'https') or not url.hostname or url.username or url.password or url.query or url.fragment or url.path:
        parser.error('--base-url must be a site origin, without credentials, a path or query')
    key = os.environ.get('SUP3API_API_KEY', '').strip()
    if (args.assets or args.models) and not key:
        parser.error('Set SUP3API_API_KEY in this shell for authenticated checks')
    opener = build_opener(NoRedirect())

    def request(path, body=None, authenticated=False, expected=200):
        headers = {'Accept': 'application/json' if path.startswith('/v1/') else '*/*'}
        if authenticated:
            headers['Authorization'] = 'Bearer ' + key
        if body is not None:
            headers['Content-Type'] = 'application/json'
        req = Request(base + path, data=json.dumps(body).encode() if body is not None else None, headers=headers)
        try:
            response = opener.open(req, timeout=25)
        except HTTPError as error:
            response = error
        with response:
            data = response.read(4 * 1024 * 1024)
            status = response.code
        allowed = expected if isinstance(expected, tuple) else (expected,)
        if status not in allowed:
            raise ValueError(f'{req.get_method()} {path}: HTTP {status}; expected {allowed}')
        print(f'PASS {req.get_method()} {path}: HTTP {status}')
        return data

    def read_json(path, **kwargs):
        return json.loads(request(path, **kwargs))

    if read_json('/health').get('status') != 'ok':
        raise ValueError('Unexpected health response')
    for path in ('/home', '/connect', '/docs/quickstart', '/docs/compatibility', '/docs/output', '/login'):
        if b'id="app"' not in request(path):
            raise ValueError(f'{path}: expected the frontend application shell')
    if b'Sup3API' not in request('/sup3api-mark.svg'):
        raise ValueError('Sup3API logo was not served')
    if read_json('/docs/assets.openapi.json').get('info', {}).get('title') != 'Sup3API Assets API':
        raise ValueError('Asset schema is missing or stale')
    if args.models:
        models = read_json('/v1/models', authenticated=True).get('data')
        if not isinstance(models, list):
            raise ValueError('Expected a model list')
        print(f'INFO {len(models)} model(s) available to this key; no inference was run')
    if args.assets:
        request('/v1/assets/capabilities', expected=(401, 403))
        providers = read_json('/v1/assets/capabilities', authenticated=True).get('providers')
        if not isinstance(providers, list):
            raise ValueError('Expected asset provider capabilities')
        tested = 0
        for provider in providers:
            if provider.get('provider') not in ('tripo', 'meshy') or not provider.get('available'):
                continue
            if 'sup3api' not in provider.get('input_formats', []):
                raise ValueError('Provider capabilities do not advertise sup3api input')
            body = {'provider': provider['provider'], 'operation': 'text_to_3d', 'input_format': 'sup3api',
                    'inputs': {'prompt': 'A low-poly wooden crate'}, 'parameters': {'texture': False, 'pbr': False}, 'output': {'formats': ['glb'], 'required_components': ['geometry']}}
            quote = read_json('/v1/assets/quotes', body=body, authenticated=True)
            if not isinstance(quote.get('quote'), dict) or quote.get('request', {}).get('provider') != provider['provider']:
                raise ValueError('Unexpected quote response')
            if not provider.get('operation_details') or not provider.get('native_api'):
                raise ValueError('Missing compatibility capabilities')
            native_base = '/providers/' + provider['provider']
            task_path = native_base + ('/v3/tasks/' if provider['provider'] == 'tripo' else '/openapi/v2/text-to-3d/') + 'sup3-smoke-foreign-task'
            request(task_path, expected=(401, 403))
            foreign = read_json(task_path, authenticated=True, expected=404)
            if foreign.get('error', {}).get('code') != 'native_task_not_found':
                raise ValueError('Native task isolation check failed')
            unsupported = native_base + ('/v3/account/balance' if provider['provider'] == 'tripo' else '/openapi/v1/balance')
            result = read_json(unsupported, authenticated=True, expected=404)
            if result.get('error', {}).get('code') != 'native_endpoint_unsupported':
                raise ValueError('Native route boundary check failed')
            tested += 1
        if not tested:
            raise ValueError('No configured 3D provider; set a server-side Tripo or Meshy API key')
        invalid = {'provider': 'meshy', 'operation': 'text_to_3d', 'input_format': 'meshy',
                   'payload': {'prompt': 'crate', 'unknown': True}}
        result = read_json('/v1/assets/quotes', body=invalid, authenticated=True, expected=400)
        if result.get('error', {}).get('code') != 'invalid_request':
            raise ValueError('Expected validation error for an unsupported native field')
        print(f'INFO {tested} provider quote(s) checked; upstream balances were not verified')
    print('PASS All requested checks completed. No generation requests were submitted.')


if __name__ == '__main__':
    try:
        main()
    except (URLError, TimeoutError):
        print('FAIL Could not reach the service; check its address and process.', file=sys.stderr)
        sys.exit(1)
    except (ValueError, KeyError, TypeError) as error:
        print('FAIL ' + str(error), file=sys.stderr)
        sys.exit(1)
