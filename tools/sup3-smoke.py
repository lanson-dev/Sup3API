"""Live Sup3API text -> rig -> animation test. Provider credits are spent only with --paid."""
import argparse
import hashlib
import json
import os
import pathlib
import time
import urllib.error
import urllib.request


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url', default='http://localhost:8080')
    parser.add_argument('--provider', choices=['tripo', 'meshy'], required=True)
    parser.add_argument('--run-id', required=True, help='Stable idempotency prefix; reuse to resume the same run')
    parser.add_argument('--output', type=pathlib.Path, default=pathlib.Path('sup3-data/smoke'))
    parser.add_argument('--paid', action='store_true', help='Allow real generation, rigging and animation charges')
    args = parser.parse_args()
    key = os.environ.get('SUP3_API_KEY')
    if not key:
        parser.error('set SUP3_API_KEY to a gateway API key')
    base = args.base_url.rstrip('/')

    def call(method, path, body=None, idem=None):
        headers = {'Authorization': 'Bearer ' + key}
        data = None
        if body is not None:
            data = json.dumps(body).encode()
            headers['Content-Type'] = 'application/json'
        if idem:
            headers['Idempotency-Key'] = idem
        req = urllib.request.Request(base + path, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=60) as response:
                return json.load(response)
        except urllib.error.HTTPError as error:
            raise RuntimeError(f'HTTP {error.code}: {error.read(2000).decode()}') from None

    def wait(job):
        deadline = time.monotonic() + 1800
        last = None
        while time.monotonic() < deadline:
            job = call('GET', '/v1/assets/jobs/' + job['id'])
            state = (job['status'], job['delivery_status'])
            if state != last:
                print(job['id'], *state, flush=True)
                last = state
            if state == ('succeeded', 'ready'):
                return job
            if job['status'] in ('failed', 'canceled', 'submission_unknown') or job['delivery_status'] == 'failed':
                raise RuntimeError(json.dumps(job.get('error') or state))
            time.sleep(5)
        raise TimeoutError('Task still running; reuse the same --run-id to resume')

    def verify(job):
        folder = args.output / job['id']
        folder.mkdir(parents=True, exist_ok=True)
        (folder / 'manifest.json').write_text(json.dumps(job, indent=2), encoding='utf-8')
        for artifact in job['artifacts']:
            req = urllib.request.Request(base + artifact['url'], headers={'Authorization': 'Bearer ' + key})
            digest = hashlib.sha256()
            size = 0
            with urllib.request.urlopen(req, timeout=120) as response:
                with (folder / (artifact['id'] + '.' + artifact['format'])).open('wb') as output:
                    while block := response.read(1024 * 1024):
                        digest.update(block)
                        size += len(block)
                        output.write(block)
            assert digest.hexdigest() == artifact['sha256'], artifact['id'] + ': SHA-256 mismatch'
            assert size == artifact['size'], artifact['id'] + ': size mismatch'
        required = ['geometry', 'materials', 'textures']
        if job['request']['operation'] in ('rig', 'animate'):
            required += ['skeleton', 'skin_weights']
        if job['request']['operation'] == 'animate':
            required += ['animations']
        for component in required:
            assert job['components'][component]['status'] == 'available', component
        print('Verified', len(job['artifacts']), 'artifacts;', job['cost'], flush=True)

    request = {
        'provider': args.provider,
        'operation': 'text_to_3d',
        'inputs': {'prompt': 'Single full body stylized adult explorer, T pose, clearly separated arms and legs, blue shirt, brown trousers and boots, front facing, no props or base.'},
        'parameters': {'texture': True, 'pbr': True, **({'target_faces': 10000, 'pose': 't-pose'} if args.provider == 'meshy' else {'max_faces': 10000})},
    }
    quote = call('POST', '/v1/assets/quotes', request)
    print('Generation quote:', quote['quote'])
    if not args.paid:
        print('Quote only. Add --paid to run generation + rig + animation with provider credits.')
        return
    for operation in ('text_to_3d', 'rig', 'animate'):
        request['operation'] = operation
        if operation != 'text_to_3d':
            request['inputs'] = {'job_id': previous['id']}
            request['parameters'] = {} if operation == 'rig' else {'animations': ['0'] if args.provider == 'meshy' else ['preset:biped:walk']}
        job = wait(call('POST', '/v1/assets/jobs', request, args.run_id + '-' + operation))
        verify(job)
        previous = job


if __name__ == '__main__':
    main()
