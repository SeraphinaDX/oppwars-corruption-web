'use strict';
// OPP Wars Web: The Go backend owns all game state. JavaScript only renders it.
const state = { game: null, panel: 'patrol', picked: 'psychic', busy: false };
const el = id => document.getElementById(id);
const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const fmt = n => Number(n || 0).toLocaleString();
const actionButton = (action, label, style = '', disabled = false) => `<button class="btn ${style}" data-action="${action}" ${disabled ? 'disabled' : ''}>${escapeHTML(label)}</button>`;
const card = (title, body) => `<section class="card"><h2>${escapeHTML(title)}</h2>${body}</section>`;
const heading = (eyebrow, title, description) => `<div class="kicker">${escapeHTML(eyebrow)}</div><h1 class="title">${escapeHTML(title)}</h1><p class="description">${escapeHTML(description)}</p>`;
const row = (label, value) => `<div class="row"><span>${escapeHTML(label)}</span><strong>${escapeHTML(value)}</strong></div>`;
const progress = (label, value, max, color) => {
  const percent = Math.max(0, Math.min(100, 100 * value / Math.max(1, max)));
  return `<div class="vital"><div class="vital-line"><span>${escapeHTML(label)}</span><strong>${fmt(value)} / ${fmt(max)}</strong></div><div class="bar"><div class="fill" style="width:${percent}%;background:${color}"></div></div></div>`;
};
const skyline = () => `<div class="scene" aria-label="Neon skyline of a troubled city"><div class="moon"></div><div class="skyline">${[55,72,42,95,63,80,45,72,54,90,69,48].map(h=>`<div class="tower" style="height:${h}%"></div>`).join('')}</div><div class="street"></div><div class="scene-badge">⚠ DISTRICT 03 // NIGHT SHIFT</div></div>`;
const activeEffect = p => p.lsdEncounters>0 ? ['REALITY DISTORTION',p.lsdEncounters,'25% counterattack dodge; 20% half-damage hit'] :
 p.fentanylEncounters>0 ? ['RECKLESS STRENGTH',p.fentanylEncounters,'+25% normal damage; +5 incoming damage'] :
 p.cocaineEncounters>0 ? ['XP RUSH',p.cocaineEncounters,'+25% XP (max +150)'] : null;
const effectPanel = p => {
 const active=activeEffect(p);
 return active?`<div class="effect-notice"><strong>◈ ${escapeHTML(active[0])}</strong><span>${active[1]} encounter${active[1]===1?'':'s'} left</span><p>${escapeHTML(active[2])}. Ends when the shift ends.</p></div>`:'';
};
const reports = g => `<section class="report"><h3>◉ DISPATCH LOG / LATEST ACTIVITY</h3>${g.messages.slice(0,6).map(m=>`<div class="report-entry">${escapeHTML(m)}</div>`).join('')}</section>`;

function setupPage() {
  return heading('◈ ACCESS TERMINAL / RECRUITMENT', 'Report for duty, Constable.', 'The city is in trouble. Your supervisor thinks you might make it worse.') +
   `<div class="card"><h2>CREATE OFFICER PROFILE</h2><label class="field">OFFICER NAME<input id="new-name" maxlength="24" placeholder="Enter your damned name" autocomplete="off" spellcheck="false"></label><div class="subhead">// CLASSIFICATION</div><div class="classes">`+
   [['robot','🤖','ROBOT','Charged Shot • builds from 1× to 4×'],['psychic','🔮','PSYCHIC','Psychic Blast • 2× damage, 4 uses per shift'],['psychopath','🔪','PSYCHOPATH','Chow Down • heal from damage dealt']].map(([key,icon,title,desc])=>`<button class="classcard ${state.picked===key?'selected':''}" data-class="${key}" aria-pressed="${state.picked===key}"><span class="glyph">${icon}</span><strong>${title}</strong><small>${desc}</small></button>`).join('')+`</div><div class="buttons">${actionButton('start','✦ BEGIN 30-DAY CAMPAIGN','primary full')}</div><div class="hint">Save data is associated with this browser session on the local Go server. No sign-up required.</div></div>`;
}
function patrolPage(g) {
  if(g.phase==='combat'){
    const e=g.enemy, p=g.player;
    return heading('◈ ACTIVE PATROL / DISTRICT 03','Street encounter','Dispatch has assigned another unfortunate situation to your shift.') + skyline() + effectPanel(p) +
      `<div class="card"><div class="enemy-header"><b>${escapeHTML(e.name.toUpperCase())}</b><span>● HOSTILE</span></div><p class="muted">Enemy health ${fmt(Math.max(0,e.hp))} / ${fmt(e.maxHP)}</p><div class="bar"><div class="fill" style="background:#f77699;width:${Math.max(0,Math.min(100,e.hp/e.maxHP*100))}%"></div></div></div>`+
      `<div class="buttons">${actionButton('fight','⚔ FIGHT','primary')}${actionButton('special',p.class==='psychic'?`✧ PSYCHIC BLAST (${Math.floor((p.psychicEnergy??100)/25)} LEFT)`:p.class==='robot'?`⚡ CHARGED SHOT (${p.classPower}×)`:'◆ CHOW DOWN','cyan',p.class==='psychic'&&(p.psychicEnergy??100)<25)}${actionButton('retreat','↳ RETURN TO HQ','full')}</div>`+reports(g);
  }
  if(g.phase==='event'){
    return heading('◈ UNEXPECTED DISPATCH / DECISION',g.event.title,'What kind of officer are you going to be?')+skyline()+card('YOUR DECISION',`<p>${escapeHTML(g.event.description)}</p><div class="buttons">${g.event.choices.map(ch=>actionButton('choose_'+ch.id,ch.label,ch.id==='renegade'?'primary':'cyan')).join('')}</div>`)+reports(g);
  }
  if(g.phase==='gameover'){
    return heading('◈ CAMPAIGN COMPLETE','Game over, Constable.',g.endReason)+card('FINAL RECORD',row('Officer',g.player.name)+row('Class',g.player.class)+row('Level',g.player.level)+row('Experience',fmt(g.player.experience))+row('Cash',`$${fmt(g.player.money)}`)+row('Paragon / Renegade',`${g.player.paragonRank} / ${g.player.renegadeRank}`)+`<div class="buttons">${actionButton('reset','↻ START NEW GAME','primary full')}</div>`)+reports(g);
  }
  return heading('◈ HQ / ACTIVE DUTY','Welcome to the night shift.','The city is falling apart. At least your paycheck is still arriving.')+skyline()+effectPanel(g.player)+
    card('NEXT ASSIGNMENT',`<p>District 03 is still dangerous. You have <strong style="color:var(--cyan)">${g.player.fightsLeft}</strong> patrol encounters remaining this shift.</p><div class="buttons">${actionButton('patrol','⚔ HIT THE STREETS','primary full',g.player.fightsLeft<=0)}${actionButton('clockout','☾ CLOCK OUT / NEXT DAY','full')}</div>`)+reports(g);
}
function hqPage(g) {
  const p=g.player;
  return heading('▣ OPERATIONS / HEADQUARTERS','Police headquarters','The coffee is stale. The paperwork is classified.')+
    card('SHIFT BOARD',row('Officer',p.name)+row('Day remaining',`${p.days} of 30`)+row('Patrol encounters remaining',p.fightsLeft)+row('Club dividends per shift',`$${fmt(Math.min(p.clubShares,5)*500)}`)+row('Base shift pay','$500')+`<div class="buttons">${actionButton('clockout','☾ CLOCK OUT & COLLECT PAY','primary full')}</div>`)+
    card('DISPATCHER NOTE','<p>Your promotions depend on experience. Reaching 10 Paragon rank may earn you powerful enemies inside the department.</p>')+reports(g);
}
function propertyPage(g) {
  const p=g.player, active=activeEffect(p);
  const items=[
    ['cocaine','Cocaine','$1,000 / gram','XP Rush: +25% XP (max +150) for 3 encounters.'],
    ['fentanyl','Fentanyl','$500 / bag','Reckless Strength: +25% normal damage but +5 retaliation damage for 2 encounters.'],
    ['lsd','LSD','$100 / five tabs','Reality Distortion: 25% chance to dodge counterattacks, but 20% chance to deal half damage for 2 encounters.']
  ];
  return heading('◫ EVIDENCE DIVISION / PROPERTY ROOM','Property room','Everything is available. Internal Affairs is unavailable.')+
    effectPanel(p)+
    card('TEMPORARY EFFECTS',`<p class="muted">Effects are activated manually at HQ, consume one item, and expire after completed patrol encounters or clock-out. Only one may run at a time. They never stack with inventory quantity.</p>`+
    items.map(([id,label,price,desc])=>{
      const count=p[id]||0;
      const units=id==='lsd'?5:1;
      return `<div class="shop-item"><div class="shop-description"><strong>${escapeHTML(label)}</strong><span class="hint">${escapeHTML(price)} · ${escapeHTML(desc)}<br>IN LOCKER: ${fmt(count)} / 20</span></div><div class="shop-actions">${actionButton('buy_'+id,'BUY','small',p.money<(id==='cocaine'?1000:id==='fentanyl'?500:100)||count+units*(p.renegadeRank>=10?2:1)>20)}${actionButton('use_'+id,'ACTIVATE','cyan small',count<1||!!active)}</div></div>`;
    }).join('')+`<p class="hint">Renegade rank 10+ doubles shop quantities, subject to the 20-unit inventory limit. These are fictional game mechanics, not real drug effects.</p>`)+reports(g);
}
function nightlifePage(g) {
  const p=g.player;
  return heading('♧ DOWNTOWN / RED ROOM','The Red Room','Bureaucracy ends where questionable entertainment begins.')+
    card('DOWNTOWN ACTIVITIES',`<p>Take an adults-only nightlife break for $100 to restore up to 200 HP (capped by level). A condom protects you from the game's infection roll; the doctor can treat infections.</p><div class="buttons">${actionButton('nightlife','♥ NIGHTLIFE BREAK • $100','primary')}${actionButton('buy_condoms','+ 5 CONDOMS • $10')}</div>`)+
    card('CLUB INVESTMENT',`<p>Own a share of the venue: $2,000 per share, providing $500 income each new shift (maximum 5 shares). This version uses venue investment rather than buying people.</p>${row('Current shares',p.clubShares)}${row('Income each shift',`$${fmt(Math.min(p.clubShares,5)*500)}`)}<div class="buttons">${actionButton('buy_club','◆ INVEST $2,000','cyan full',p.clubShares>=5)}</div>`)+reports(g);
}
function doctorPage(g) {
  const p=g.player;
  return heading('✚ MEDICAL SERVICES / CLINIC','Doctors office','The doctor refuses to ask how you got here.')+
    card('MEDICAL RECORD',row('Current HP',fmt(p.hp))+row('Active infections',p.infections)+row('Cure cost','$100')+`<div class="buttons">${actionButton('clinic','✚ TREAT INFECTIONS • $100','cyan full',p.infections===0)}</div><p class="hint">Infections reduce daily patrol capacity and cause a $25 per-infection income penalty when you clock out.</p>`)+reports(g);
}
function recordPage(g) {
 const p=g.player;
 return heading('▤ PERSONNEL / CLASSIFIED DOSSIER','Officer file','Everything on record. Nothing off the record.')+
   card('PERSONNEL',row('Name',p.name)+row('Classification',p.class)+row('Level',p.level)+row('Attack',p.attack)+row(p.class==='robot'?'Charged Shot multiplier':p.class==='psychic'?'Psychic energy':'Special ability',p.class==='robot'?`${p.classPower}×`:p.class==='psychic'?`${p.psychicEnergy??100} / 100`:'Chow Down')+row('Experience',fmt(p.experience))+row('Paragon',p.paragonRank)+row('Renegade',p.renegadeRank)+row('Remaining days',p.days))+
   card('INVENTORY',row('Cocaine',p.cocaine)+row('Fentanyl',p.fentanyl)+row('LSD',p.lsd)+row('Condoms',p.condoms)+row('Club shares',p.clubShares)+row('Active status',activeEffect(p)?.[0]||'None')+row('Status encounters left',activeEffect(p)?.[1]||0))+
   card('PERSONNEL ACTIONS',`<p>Starting a new campaign replaces this browser session's current save.</p><div class="buttons">${actionButton('reset','↻ NEW GAME','alert full')}</div>`)+reports(g);
}
const pHasLSD = g => g.phase!=='setup' && g.player.lsdEncounters>0;
function render() {
 const g=state.game;
 if(!g) return;
 let page;
 if(g.phase==='setup') page=setupPage();
 else if(g.phase!=='hq') page=patrolPage(g);
 else page=({patrol:patrolPage,hq:hqPage,property:propertyPage,nightlife:nightlifePage,doctor:doctorPage,record:recordPage}[state.panel] || patrolPage)(g);
 el('content').innerHTML=page;
 el('app').classList.toggle('lsd-active',pHasLSD(g));
 document.querySelectorAll('.nav-btn').forEach(b=>b.classList.toggle('active',b.dataset.panel===state.panel));
 document.querySelectorAll('.nav-btn').forEach(b=>b.disabled=g.phase!=='hq');
 const p=g.player;
 el('officer-name').textContent=p.name || 'UNASSIGNED';
 el('officer-class').textContent=p.class ? `${p.class.toUpperCase()} // LEVEL ${p.level}` : 'CLASS NOT SELECTED';
 el('shift-status').textContent=g.phase==='setup'?'RECRUITMENT OPEN':g.phase==='gameover'?'CAMPAIGN FINISHED':g.phase==='combat'?'COMBAT ACTIVE':`${p.days} DAYS REMAINING`;
 if(g.phase==='setup') {el('vitals').innerHTML='';el('brief').innerHTML='<p class="hint">Choose a class to get started.</p>';return}
 const xpMax=([0,...[1000,4000,9000,18000,30000,60000,120000,240000,480000,960000,1920000,3840000,7680000,15360000]][p.level] || 15360000);
 el('vitals').innerHTML=progress('HEALTH',p.hp,Math.min(720,200+Math.max(0,p.level-1)*40),'#6addab')+progress('EXPERIENCE',p.experience,xpMax,'#ffc778')+(p.class==='psychic'?progress('PSYCHIC ENERGY',p.psychicEnergy??100,100,'#8a80ff'):p.class==='robot'?progress('CHARGED SHOT',p.classPower,4,'#57e8ef'):'')+
  `<div class="vital"><div class="vital-line"><span>KARMA</span><strong style="color:var(--pink)">${p.renegadeRank>=p.paragonRank?'RENEGADE':'PARAGON'}</strong></div><div class="bar"><div class="fill" style="width:${Math.min(100,Math.abs(p.renegadeRank-p.paragonRank)*10)}%;background:var(--pink)"></div></div></div>`;
 el('brief').innerHTML=row('CASH',`$${fmt(p.money)}`)+row('PATROLS',`${p.fightsLeft} / ${25}`)+row('DAYS',p.days)+row('INFECTIONS',p.infections)+row('ACTIVE EFFECT',activeEffect(p)?.[0]||'None');
}
function toast(msg){const box=el('toast');box.textContent=msg;box.classList.add('visible');clearTimeout(toast.timer);toast.timer=setTimeout(()=>box.classList.remove('visible'),4100)}
async function request(action,name='',officerClass=''){
 if(state.busy)return;
 state.busy=true;
 try{
  const response=await fetch('/api/action',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify({action,name,class:officerClass})});
  const data=await response.json();
  if(!response.ok)throw Error(data.error||'server error');
  state.game=data;state.panel='patrol';render();
 }catch(err){toast(err.message || 'Connection lost')}
 finally{state.busy=false}
}
document.addEventListener('click',e=>{
 const cls=e.target.closest('[data-class]');if(cls){state.picked=cls.dataset.class;const name=el('new-name')?.value;render();if(el('new-name'))el('new-name').value=name||'';return}
 const nav=e.target.closest('[data-panel]');if(nav && state.game?.phase==='hq'){state.panel=nav.dataset.panel;render();return}
 const btn=e.target.closest('[data-action]');if(!btn)return;
 if(btn.dataset.action==='reset'&&!confirm('Replace this browser session save with a new game?'))return;
 request(btn.dataset.action,btn.dataset.action==='start'?(el('new-name')?.value||''):'',state.picked);
});
(async()=>{
 try{const res=await fetch('/api/state',{cache:'no-store'});if(!res.ok)throw Error('Cannot connect to OPP Wars server');state.game=await res.json();render()}
 catch(e){el('content').innerHTML=heading('◈ OFFLINE','Connection failed','The server did not answer. Reload after starting the Go application.');toast(e.message)}
})();
