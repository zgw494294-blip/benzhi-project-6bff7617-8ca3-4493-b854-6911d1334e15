const form=document.querySelector('#project');
const filters=document.querySelector('#filters');
const list=document.querySelector('#project-list');
const out=document.querySelector('#output');
async function json(url,options){const response=await fetch(url,options);const body=await response.json().catch(()=>({error:'响应不是有效 JSON'}));if(!response.ok)throw body;return body}
function show(value){out.textContent=JSON.stringify(value,null,2)}
async function loadProjects(){const query=new URLSearchParams(new FormData(filters));for(const [key,value] of [...query])if(!value)query.delete(key);try{const body=await json(`/api/projects?${query}`);list.replaceChildren(...body.projects.map(project=>{const button=document.createElement('button');button.type='button';button.className='project-row';button.textContent=`${project.title} · ${project.status} · v${project.version} · 片段 ${project.segmentCount} · 修订 ${project.revisionCount} · 未关闭 ${project.openIssueCount}`;button.addEventListener('click',()=>loadDetail(project.projectID));return button}));if(!body.projects.length)list.textContent='没有符合筛选条件的项目'}catch(error){show(error)}}
async function loadDetail(projectID){try{show(await json(`/api/projects/${encodeURIComponent(projectID)}`))}catch(error){show(error)}}
filters.addEventListener('submit',event=>{event.preventDefault();loadProjects()});
form.addEventListener('submit',async event=>{event.preventDefault();const data=Object.fromEntries(new FormData(form));data.collectionSites=data.collectionSites.split(',').map(value=>value.trim()).filter(Boolean);data.consentRefs=data.consentRefs.split(',').map(value=>value.trim()).filter(Boolean);data.expectedVersion=0;data.idempotencyKey=crypto.randomUUID();try{const result=await json('/api/projects',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(data)});show(result);form.reset();await loadProjects()}catch(error){show(error)}});
loadProjects();
