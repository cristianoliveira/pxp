// Standalone interaction sketch. No server calls or agent decisions.
const reference = '../../tests/smoke/fixtures/image-diff/real-ui-reference.png';
const actual = '../../tests/smoke/fixtures/image-diff/real-ui-implementation.png';
const mode = document.body.dataset.mode;
const concepts = {desk:['Review desk','Compare first. Keep feedback beside the evidence.'],focus:['Focus mode','One canvas. More room to inspect the details.'],rounds:['Round tracker','See what changed before asking for the next fix.']};
const [title, description] = concepts[mode];
document.body.className = mode;
document.body.innerHTML = `
<header><div><div class="eyebrow">pxp / human review</div><h1>${title}</h1><p class="muted">${description}</p></div><span class="badge" id="waiting">● Agent waiting for your review</span></header>
<nav aria-label="UX concepts">${Object.entries(concepts).map(([key,value])=>`<a href="${key}.html" ${mode===key?'aria-current="page"':''}>${value[0]}</a>`).join('')}<span class="prototype">INTERACTIVE MOCK · no feedback is sent to an agent</span></nav>
<main class="layout">
${mode==='rounds'?`<aside class="card section history"><div class="eyebrow">Example history</div><h2>Review rounds</h2><ol class="timeline"><li><h3>Round 1</h3><small>Feedback submitted</small><p>2 changes requested</p></li><li class="current"><h3>Round 2</h3><small>Awaiting your review</small></li></ol><p class="muted">Earlier screenshots and notes stay with their original round.</p></aside>`:''}
<section class="card"><div class="toolbar"><div><strong>${mode==='rounds'?'Round 2':'Round 1'}</strong><div class="muted">App lock settings · immutable snapshot</div></div><div class="tools" aria-label="Image view">${(mode==='desk'?['Compare','Overlay']:['Current','Reference','Overlay']).map((v,i)=>`<button type="button" data-view="${v}" aria-pressed="${i===0}">${v}</button>`).join('')}</div></div>
${mode==='rounds'?`<div class="summary"><div class="eyebrow">Illustrative agent update</div><p>“Adjusted panel spacing and the primary action.”</p><span class="resolved">Agent claims 2 notes addressed · verify below</span><p class="muted">Example copy only; these fixtures are not a real second round.</p></div>`:''}
<div class="images"><figure><figcaption>REFERENCE <span class="muted">Target</span></figcaption><div class="shot"><img src="${reference}" alt="Reference app lock settings"></div></figure><figure><figcaption><span id="view-label">CURRENT</span><span class="muted">Click to add a pin</span></figcaption><div class="shot annotate" id="canvas"><img id="current" src="${actual}" alt="Current app lock settings"><img class="overlay-img" src="${reference}" alt=""></div></figure></div>
<div class="hint">Pin: click current image, then enter a note. Overlay is a blended preview, not a mismatch mask. Rectangle drawing is outside this sketch.</div></section>
<aside class="card section"><div class="eyebrow">Your feedback</div><h2>What needs to change?</h2><div id="notes-list"><p class="empty">No pins yet. Click the current image to point to a detail.</p></div><label for="general">General note</label><textarea id="general" placeholder="Describe the desired result…"></textarea><p class="muted" id="draft">Draft saved locally in this browser.</p><div class="actions"><button type="button" class="primary" id="submit">Submit feedback →</button><small>Hand notes to the agent for another round.</small><button type="button" id="approve">Approve this round</button><small>Finish review without requesting changes.</small></div><p class="status" role="status" id="status"></p></aside>
</main><footer class="bottom">Closing this tab is not approval. Reopen this page to resume your draft. In the real loop, submitting starts agent work; approval ends review.</footer>`;
const key = `pxp-ux-mock-${mode}`;
let pins = [];
const general = document.querySelector('#general');
const status = document.querySelector('#status');
try { const saved = JSON.parse(localStorage.getItem(key) || '{}'); pins = Array.isArray(saved.pins) ? saved.pins : []; general.value = saved.general || ''; } catch { document.querySelector('#draft').textContent = 'Local storage unavailable; keep this tab open.'; }
function save() { try { localStorage.setItem(key, JSON.stringify({pins, general:general.value})); } catch { document.querySelector('#draft').textContent = 'Draft could not be saved; keep this tab open.'; } }
function render() {
 const list = document.querySelector('#notes-list');
 list.replaceChildren();
 document.querySelectorAll('.pin').forEach(pin=>pin.remove());
 if (!pins.length) { const empty=document.createElement('p'); empty.className='empty'; empty.textContent='No pins yet. Click the current image to point to a detail.'; list.append(empty); }
 pins.forEach((pin,i)=>{
  const row=document.createElement('div'); row.className='note';
  const remove=document.createElement('button'); remove.type='button'; remove.textContent='Remove'; remove.setAttribute('aria-label',`Remove pin ${i+1}`); remove.onclick=()=>{pins.splice(i,1); save(); render();};
  const text=document.createElement('span'); text.textContent=`${i+1}. ${pin.text}`; row.append(remove,text); list.append(row);
  const dot=document.createElement('span'); dot.className='pin'; dot.textContent=i+1; dot.style.left=`${pin.x}%`; dot.style.top=`${pin.y}%`; document.querySelector('#canvas').append(dot);
 });
}
general.addEventListener('input',save);
document.querySelector('#canvas').addEventListener('click',event=>{
 if(document.querySelector('#current').getAttribute('src')!==actual || document.querySelector('.images').classList.contains('overlay')) {status.textContent='Switch to Current to add a pin.';return;}
 const bounds=event.currentTarget.getBoundingClientRect();
 const text=prompt('What should change at this point?'); if(!text?.trim()) return;
 pins.push({x:(event.clientX-bounds.left)/bounds.width*100,y:(event.clientY-bounds.top)/bounds.height*100,text:text.trim()}); save();render();
});
document.querySelectorAll('[data-view]').forEach(button=>button.onclick=()=>{
 document.querySelectorAll('[data-view]').forEach(b=>b.setAttribute('aria-pressed',String(b===button)));
 const view=button.dataset.view;
 document.querySelector('.images').classList.toggle('overlay',view==='Overlay');
 document.querySelector('#current').src=view==='Reference'?reference:actual;
 document.querySelector('#current').alt=view==='Reference'?'Reference app lock settings':'Current app lock settings';
 document.querySelector('#view-label').textContent=view==='Compare'?'CURRENT':view.toUpperCase();
 document.querySelectorAll('.pin').forEach(pin=>pin.hidden=view==='Reference');
});
document.querySelector('#submit').onclick=()=>{
 if(!general.value.trim()&&!pins.length){status.textContent='Add a note before submitting. The agent is still waiting.';return;}
 status.textContent='Preview: feedback submitted → agent works → a new round opens. No real agent was notified.';
};
document.querySelector('#approve').onclick=()=>{
 if(general.value.trim()||pins.length){status.textContent='You have draft feedback. Submit it, or clear it before approving.';return;}
 status.textContent='Preview: round approved → review complete. No real decision was sent.';
};
render();
