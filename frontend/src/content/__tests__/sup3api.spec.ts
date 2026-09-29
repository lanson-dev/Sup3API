import { describe, it, expect } from 'vitest'
import { buildRequest, curlExample, sdkExample, docs, type BuildInput } from '../sup3api'

const base: BuildInput = { protocol:'responses', model:'test-model', prompt:'Describe this image', image:'https://example.com/a.png', operation:'image_to_3d', images:[], source:'job_owned', texture:true, pbr:true, animation:'0,1', native:false }

describe('Sup3API protocol examples', () => {
  it('keeps each provider image content block in its own protocol', () => {
    expect(buildRequest(base)).toMatchObject({ input:[{ content:[{type:'input_image',image_url:base.image},{type:'input_text'}] }] })
    expect(buildRequest({...base,protocol:'chat'})).toMatchObject({ messages:[{content:[{type:'image_url',image_url:{url:base.image}},{type:'text'}]}] })
    expect(buildRequest({...base,protocol:'claude',image:'data:image/png;base64,YQ=='})).toMatchObject({ max_tokens:1024,messages:[{content:[{type:'image',source:{type:'base64',media_type:'image/png',data:'YQ=='}},{type:'text'}]}] })
  })
  it('does not add a paid Meshy refine to native preview', () => {
    const request=buildRequest({...base,protocol:'meshy',model:'meshy-7.1',operation:'text_to_3d',native:true})
    expect(request).toEqual({provider:'meshy',operation:'text_to_3d',input_format:'meshy',payload:{ai_model:'meshy-7.1',mode:'preview',prompt:base.prompt}})
  })
  it('uses owned gateway task IDs and preserves animation IDs', () => {
    expect(buildRequest({...base,protocol:'meshy',model:'',operation:'animate',native:true})).toEqual({provider:'meshy',operation:'animate',input_format:'meshy',payload:{rig_task_id:'job_owned',action_ids:[0,1]}})
    expect(buildRequest({...base,protocol:'tripo',operation:'retexture',source:'https://example.com/model.glb'})).toMatchObject({inputs:{model_url:'https://example.com/model.glb',prompt:base.prompt}})
  })
  it('preserves empty Tripo view slots and never adds PBR to untextured requests', () => {
    expect(buildRequest({...base,protocol:'tripo',operation:'multi_image_to_3d',images:[base.image,'',base.image,''],texture:false})).toMatchObject({inputs:{images:[base.image,'',base.image,'']},parameters:{texture:false,pbr:false}})
  })
  it('uses environment keys and preserves quotes/newlines in generated examples', () => {
    const body=buildRequest({...base,prompt:'A "box"\nwith a \'handle\' and $HOME'})
    const curl=curlExample('/v1/assets/jobs',body)
    expect(curl).toContain('Idempotency-Key: $JOB_REQUEST_ID')
    expect(curl).toContain("<<'SUP3API_JSON'")
    expect(curl).toContain(JSON.stringify(body,null,2))
    expect(sdkExample('/v1/messages',body,'javascript')).toContain('anthropic-version')
    expect(sdkExample('/v1/responses',body,'python')).toContain('json=json.loads(')
  })
  it('documents every linked internal article and full result boundaries', () => {
    const ids=new Set(docs.map(d=>d.id));expect(ids.size).toBe(docs.length)
    for(const doc of docs)for(const section of doc.sections)for(const link of section.links||[]) {
      if(link.href.startsWith('/docs/')&&!link.href.endsWith('.json'))expect(ids.has(link.href.slice(6))).toBe(true)
    }
    expect(JSON.stringify(docs)).toContain('steps[].provider_result')
    expect(JSON.stringify(docs)).toContain('submission_unknown')
  })
})
