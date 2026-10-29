'use strict';
// OPP Wars Web: The Go backend owns all game state. JavaScript only renders it.
const state = { game: null, panel: 'patrol', picked: 'psychic', busy: false, auth: null, precinct: null, viewingOfficer: null, calendarBusy: false, lastCalendarCheck: 0 };
const el = id => document.getElementById(id);
const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const fmt = n => Number(n || 0).toLocaleString();
const shopPrices = Object.freeze({cocaine:400, fentanyl:150, lsd:75}); // Mirrors backend balance.go
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
const utcResetTime = g => g?.lastResetDate ? Date.parse(g.lastResetDate+'T00:00:00Z')+86400000 : 0;
function resetCountdown() {
 const target=utcResetTime(state.game), targetEl=el('daily-reset');
 if(!targetEl)return;
 if(!target){targetEl.textContent='UTC CALENDAR NOT STARTED';return}
 const seconds=Math.max(0,Math.ceil((target-Date.now())/1000));
 const hrs=Math.floor(seconds/3600), mins=Math.floor(seconds%3600/60), secs=seconds%60;
 targetEl.textContent=`NEXT UTC RESET • ${String(hrs).padStart(2,'0')}:${String(mins).padStart(2,'0')}:${String(secs).padStart(2,'0')}`;
}
async function checkCalendar() {
 const g=state.game;
 if(!g||g.phase==='setup'||g.phase==='gameover'||state.calendarBusy||!utcResetTime(g)||Date.now()<utcResetTime(g)) return;
 if(Date.now()-state.lastCalendarCheck<15000)return;
 state.calendarBusy=true;state.lastCalendarCheck=Date.now();
 try{
  const res=await fetch('/api/state',{cache:'no-store'});
  if(!res.ok)throw Error('Daily reset unavailable');
  state.game=await res.json();
  state.panel='patrol';state.viewingOfficer=null;
  render();
 }catch(err){toast(err.message)}
 finally{state.calendarBusy=false}
}
const reports = g => `<section class="report"><h3>◉ DISPATCH LOG / LATEST ACTIVITY</h3>${g.messages.slice(0,6).map(m=>`<div class="report-entry">${escapeHTML(m)}</div>`).join('')}</section>`;

function setupPage() {
  return heading('◈ ACCESS TERMINAL / RECRUITMENT', 'Report for duty, Constable.', 'The city is in trouble. Your 30 real-life duty days begin today and reset at midnight UTC.') +
   `<div class="card"><h2>CREATE OFFICER PROFILE</h2><label class="field">OFFICER NAME<input id="new-name" maxlength="24" placeholder="Enter your damned name" autocomplete="off" spellcheck="false"></label><div class="subhead">// CLASSIFICATION</div><div class="classes">`+
   [['robot','🤖','ROBOT','Charged Shot • builds from 1× to 4×'],['psychic','🔮','PSYCHIC','Psychic Blast • 2× damage, 12 uses per shift'],['psychopath','🔪','PSYCHOPATH','Chow Down • heal from damage dealt']].map(([key,icon,title,desc])=>`<button class="classcard ${state.picked===key?'selected':''}" data-class="${key}" aria-pressed="${state.picked===key}"><span class="glyph">${icon}</span><strong>${title}</strong><small>${desc}</small></button>`).join('')+`</div><div class="buttons">${actionButton('start','✦ BEGIN 30-DAY CAMPAIGN','primary full')}</div><div class="hint">Save data is associated with this browser session on the local Go server. No sign-up required.</div></div>`;
}
function patrolPage(g) {
  if(g.phase==='combat'){
    const e=g.enemy, p=g.player;
    return heading('◈ ACTIVE PATROL / DISTRICT 03',`${p.name} vs. ${e.name}`,'Dispatch has assigned another unfortunate situation to your shift.') + skyline() + effectPanel(p) +
      `<div class="card"><div class="enemy-header"><b>${escapeHTML(e.name.toUpperCase())}</b><span>● HOSTILE</span></div><p class="muted">Enemy health ${fmt(Math.max(0,e.hp))} / ${fmt(e.maxHP)}</p><div class="bar" role="progressbar" aria-label="Enemy health" aria-valuemin="0" aria-valuemax="${e.maxHP}" aria-valuenow="${Math.max(0,e.hp)}"><div class="fill" style="background:#f77699;width:${Math.max(0,Math.min(100,e.hp/e.maxHP*100))}%"></div></div>${e.catchphrase?`<blockquote class="enemy-quote">“${escapeHTML(e.catchphrase)}”</blockquote>`:''}</div>`+
      `<p class="hint">Arrest a suspect below 50% HP with one pair of hand cuffs. Arrests earn Paragon but pay less than eliminations.</p><div class="buttons">${actionButton('fight','⚔ FIGHT','primary')}${actionButton('special',p.class==='psychic'?`✧ PSYCHIC BLAST (${Math.floor((p.psychicEnergy??300)/25)} LEFT)`:p.class==='robot'?`⚡ CHARGED SHOT (${p.classPower}×)`:'◆ CHOW DOWN','cyan',p.class==='psychic'&&(p.psychicEnergy??100)<25)}${actionButton('arrest',`♙ ARREST (${fmt(p.handcuffs || 0)} CUFFS)`,'cyan',!p.handcuffs || e.hp<=0 || e.hp>e.maxHP*0.5)}${actionButton('retreat','↳ RETURN TO HQ')}</div>`+reports(g);
  }
  if(g.phase==='event'){
    return heading('◈ UNEXPECTED DISPATCH / DECISION',g.event.title,'What kind of officer are you going to be?')+skyline()+card('YOUR DECISION',`<p>${escapeHTML(g.event.description)}</p><div class="buttons">${g.event.choices.map(ch=>actionButton('choose_'+ch.id,ch.label,ch.id==='renegade'?'primary':'cyan')).join('')}</div>`)+reports(g);
  }
  if(g.phase==='gameover'){
    return heading('◈ CAMPAIGN COMPLETE','Game over, Constable.',g.endReason)+card('FINAL RECORD',row('Officer',g.player.name)+row('Class',g.player.class)+row('Level',g.player.level)+row('Experience',fmt(g.player.experience))+row('Cash',`$${fmt(g.player.money)}`)+row('Paragon / Renegade',`${g.player.paragonRank} / ${g.player.renegadeRank}`)+`<div class="buttons">${actionButton('reset','↻ START NEW GAME','primary full')}</div>`)+reports(g);
  }
  return heading('◈ HQ / ACTIVE DUTY','Welcome to the night shift.','Your patrol turns reset at midnight UTC. Today is one of your 30 real-world campaign days.')+skyline()+effectPanel(g.player)+
    card('NEXT ASSIGNMENT',`<p>District 03 is still dangerous. You have <strong style="color:var(--cyan)">${g.player.fightsLeft}</strong> patrol encounters remaining today.</p><p class="hint">Your turns refresh at midnight UTC. Clocking out ends today's shift, not the calendar day.</p><div class="buttons">${actionButton('patrol','⚔ HIT THE STREETS','primary full',g.player.fightsLeft<=0)}${actionButton('clockout','☾ CLOCK OUT FOR TODAY','full')}</div>`)+reports(g);
}
function shiftDonePage(g) {
 return heading('◈ HQ / SHIFT COMPLETE','Off duty until midnight UTC',`${g.player.name} has finished today's shift. No more patrols or purchases until the next UTC day.`)+
  card('TODAY IS CLOSED',row('Campaign day',`${31-g.player.days} / 30`)+row('Days remaining',g.player.days)+row('Next daily reset','00:00 UTC')+
   '<p class="hint">The precinct roster, public officer profiles, and your personnel file remain available while off duty.</p>')+
  reports(g);
}
function hqPage(g) {
  const p=g.player;
  return heading('▣ OPERATIONS / HEADQUARTERS','Police headquarters','The coffee is stale. The paperwork is classified.')+
    card('SHIFT BOARD',row('Officer',p.name)+row('Calendar day',`${31-p.days} of 30`)+row('Days remaining (including today)',p.days)+row('Patrol turns remaining',p.fightsLeft)+row('Campaign started (UTC)',g.campaignStart||'Unassigned')+row('Club dividends per shift',`$${fmt(Math.min(p.clubShares,5)*500)}`)+row('Base shift pay','$500')+`<div class="buttons">${actionButton('clockout','☾ CLOCK OUT & COLLECT PAY','primary full')}</div>`)+
    card('DISPATCHER NOTE','<p>Your promotions depend on experience. Reaching 100 Paragon rank may earn you powerful enemies inside the department.</p>')+reports(g);
}
function propertyPage(g) {
  const p=g.player, active=activeEffect(p);
  const items=[
    ['cocaine','Cocaine',`${fmt(shopPrices.cocaine)} / gram`,'XP Rush: +25% XP (max +150) for 3 encounters.'],
    ['fentanyl','Fentanyl',`${fmt(shopPrices.fentanyl)} / bag`,'Reckless Strength: +25% normal damage but +5 retaliation damage for 2 encounters.'],
    ['lsd','LSD',`${fmt(shopPrices.lsd)} / tab`,'Reality Distortion: 25% chance to dodge counterattacks, but 20% chance to deal half damage for 2 encounters.']
  ];
  return heading('◫ EVIDENCE DIVISION / PROPERTY ROOM','Property room','Everything is available. Internal Affairs is unavailable.')+
    effectPanel(p)+
    card('TEMPORARY EFFECTS',`<p class="muted">Effects are activated manually at HQ, consume one item, and expire after completed patrol encounters or clock-out. Only one may run at a time. They never stack with inventory quantity.</p>`+
    items.map(([id,label,price,desc])=>{
      const count=p[id]||0;
      const units=1;
      return `<div class="shop-item"><div class="shop-description"><strong>${escapeHTML(label)}</strong><span class="hint">${escapeHTML(price)} · ${escapeHTML(desc)}<br>IN LOCKER: ${fmt(count)} / 20</span></div><div class="shop-actions">${actionButton('buy_'+id,'BUY','small',p.money<shopPrices[id]||count+units*(p.renegadeRank>=100?2:1)>20)}${actionButton('use_'+id,'ACTIVATE','cyan small',count<1||!!active)}</div></div>`;
    }).join('')+`<p class="hint">Renegade rank 100+ doubles shop quantities, subject to the 20-unit inventory limit. These are fictional game mechanics, not real drug effects.</p>`)+
    card('ARREST EQUIPMENT',`<div class="shop-item"><div class="shop-description"><strong>Hand cuffs</strong><span class="hint">$100 per pair. Each successful arrest uses one pair and awards +1 Paragon. Suspects must be below 50% HP.<br>IN LOCKER: ${fmt(p.handcuffs || 0)} / 20</span></div><div class="shop-actions">${actionButton('buy_handcuffs','BUY • $100','small',p.money<100||(p.handcuffs||0)+(p.renegadeRank>=100?2:1)>20)}</div></div>`)+reports(g);
}
function redRoomResultPage(g) {
 const r=g.redRoom;
 if(!r) return heading('♧ RED ROOM / REPORT MISSING','Report unavailable','Return to the main menu and try again.');
 const headingText = r.infectionGained ? 'UNWELCOME SOUVENIR' : 'NO NEW INFECTION';
 const report = r.infectionGained
   ? 'The morning clinic report is in: you picked up an infection. The doctor will be expecting you.'
   : r.protected
     ? 'You used a condom. No new infection was recorded during this encounter.'
     : 'You got lucky this time. No new infection was recorded during this encounter.';
 const tone = r.infectionGained ? 'bad' : 'good';
 return heading('♧ RED ROOM / AFTER-ACTION REPORT','A night at the Red Room','The curtains close, the neon flickers, and the story does not end there.') +
   card('THE ENCOUNTER',`<p class="redroom-story">${escapeHTML(r.scene)}</p>`) +
   `<section class="redroom-result ${tone}" aria-label="Encounter outcome"><div class="kicker">// MEDICAL REPORT</div><h2>${headingText}</h2><p>${escapeHTML(report)}</p>${row('Current infections',r.totalInfections)}${row('Condom used',r.protected?'Yes':'No')}${row('Health restored',`+${fmt(r.healthRecovered)} HP`)+(g.player.class==='psychic'?row('Psychic energy restored',`+${fmt(r.psychicEnergyRestored??0)} PSI`):'')+row('Patrol turns spent',r.turnsSpent??0)}${row('Money spent',`$${fmt(r.cost)}`)}</section>` +
   `<div class="buttons">${actionButton('continue','↳ CONTINUE','primary full')}</div>`;
}
function nightlifePage(g) {
  const p=g.player;
  return heading('♧ DOWNTOWN / RED ROOM','The Red Room','The neon glows, the music pounds, and nobody wants to see your badge.')+
    card('AFTER HOURS',`<p>The Red Room has adult company for hire. A private encounter costs $100. A visit costs one patrol turn. Psychics restore up to six Psychic Blasts per visit, capped at twelve. The house sells condoms at the bar, and the clinic opens early for anyone who needs it.</p><div class="buttons">${actionButton('nightlife','♥ BANG A PROSTITUTE • $100 + 1 TURN','primary',p.money<100||p.fightsLeft<=0)}${actionButton('buy_condoms','+ 5 CONDOMS • $250','',p.money<250)}</div>`)+
    card('CLUB INVESTMENT',`<p>The manager is offering a slice of the business for $2,000. Shareholders get a cut of the takings every shift, assuming the manager doesn't run off with the till.</p>${row('Your club shares',p.clubShares)}${row('Your shift dividend',`$${fmt(Math.min(p.clubShares,5)*500)}`)}<div class="buttons">${actionButton('buy_club','◆ BUY A SHARE • $2,000','cyan full',p.clubShares>=5||p.money<2000)}</div>`)+reports(g);
}
function doctorPage(g) {
  const p=g.player;
  const cost = Math.max(0,p.infections)*500;
  const damagePenalty = Math.min(45,Math.max(0,p.infections)*15);
  const nextPatrols = p.infections>0 ? Math.max(5,25-15-p.infections*3) : 25;
  return heading('✚ MEDICAL SERVICES / CLINIC','Doctors office','The doctor has seen worse. The bill may hurt more than the diagnosis.')+
    card('MEDICAL RECORD',row('Current HP',fmt(p.hp))+row('Active infections',p.infections)+row('Damage penalty',p.infections?`-${damagePenalty}%`:'None')+row('Pay deducted per shift',`$${fmt(p.infections*150)}`)+row('Patrol slots next shift',nextPatrols)+row('Treatment bill',`$${fmt(cost)}`)+`<div class="buttons">${actionButton('clinic',`✚ TREAT INFECTIONS • $${fmt(cost)}`,'cyan full',p.infections===0||p.money<cost)}</div><p class="hint">The clinic clears every infection on your record. While sick, your attacks hit softer, your patrol assignments dwindle and the department docks your pay.</p>`)+reports(g);
}
function recordPage(g) {
 const p=g.player;
 return heading('▤ PERSONNEL / CLASSIFIED DOSSIER','Officer file','Everything on record. Nothing off the record.')+
   card('PERSONNEL',row('Name',p.name)+row('Classification',p.class)+row('Level',p.level)+row('Attack',p.attack)+row(p.class==='robot'?'Charged Shot multiplier':p.class==='psychic'?'Psychic energy':'Special ability',p.class==='robot'?`${p.classPower}×`:p.class==='psychic'?`${p.psychicEnergy??300} / 300`:'Chow Down')+row('Experience',fmt(p.experience))+row('Paragon',p.paragonRank)+row('Renegade',p.renegadeRank)+row('Remaining days',p.days)+row('Combat illness penalty',p.infections?`-${Math.min(45,p.infections*15)}% damage`:'None'))+
   card('INVENTORY',row('Cocaine',p.cocaine)+row('Fentanyl',p.fentanyl)+row('LSD',p.lsd)+row('Condoms',p.condoms)+row('Hand cuffs',p.handcuffs||0)+row('Club shares',p.clubShares)+row('Active status',activeEffect(p)?.[0]||'None')+row('Status encounters left',activeEffect(p)?.[1]||0))+
   card('PERSONNEL ACTIONS',`<p>Starting a new campaign replaces this browser session's current save.</p><div class="buttons">${actionButton('reset','↻ NEW GAME','alert full')}</div>`)+reports(g);
}
function authPage() {
 const username=state.auth?.username;
 if(username) return heading('◈ ACCOUNT / SIGNED IN','Officer account','Your character is bound to this persistent online account.')+
  card('ACCOUNT STATUS',row('Username',username)+row('Game save','Stored under your account')+
   `<div class="buttons"><button class="btn full" data-logout="true">SIGN OUT</button></div><p class="hint">Signing out leaves your officer saved on the server. Sign in from another browser to resume.</p>`);
 return heading('◈ PRECINCT / ACCESS CONTROL','Join the online precinct','Create an account to keep your officer and appear to other players.')+
  card('CREATE ACCOUNT',`<form class="account-form" data-auth-form="register"><label class="field">USERNAME<input name="username" required minlength="3" maxlength="24" pattern="[A-Za-z][A-Za-z0-9_]{2,23}" autocomplete="username" placeholder="Officer handle"></label><label class="field">PASSWORD<input name="password" type="password" required minlength="12" maxlength="128" autocomplete="new-password" placeholder="At least 12 characters"></label><button class="btn primary full" type="submit">CREATE ACCOUNT & SAVE OFFICER</button></form><p class="hint">Your current guest campaign becomes your account's officer. Usernames cannot be changed yet.</p>`)+
  card('RETURNING OFFICER',`<form class="account-form" data-auth-form="login"><label class="field">USERNAME<input name="username" required autocomplete="username"></label><label class="field">PASSWORD<input name="password" type="password" required autocomplete="current-password"></label><button class="btn cyan full" type="submit">SIGN IN</button></form><p class="hint">Sign in to retrieve your existing character. A guest save in this browser will not be merged into an existing account.</p>`);
}
const officerDisplay = p => `<span class="presence ${p.online?'online':'offline'}">${p.online?'● ONLINE':'○ OFFLINE'}</span> <strong>${escapeHTML(p.username)}</strong> <span class="muted">${escapeHTML(p.officerName || 'Unassigned')} • ${escapeHTML(p.class || 'Recruit')} Lv. ${fmt(p.level)}</span>`;
function precinctPage(g) {
 if(!state.auth?.authenticated) return heading('◈ DISTRICT / SHARED PRECINCT','Online precinct','The doors are open to registered officers.')+
   card('ACCOUNT REQUIRED',`<p>Register or sign in to see who is on duty, inspect officer profiles and compare experience rankings.</p><div class="buttons"><button class="btn primary full" data-panel="account">◈ OPEN ACCOUNT TERMINAL</button></div>`);
 if(state.viewingOfficer){
  const p=state.viewingOfficer;
  return heading('◈ PRECINCT / PERSONNEL','Public officer file','Available to everyone signed into the precinct.')+
   card('OFFICER PROFILE',row('Account',p.username)+row('Officer',p.officerName||'Unassigned')+row('Status',p.online?'Online':'Offline')+row('Class',p.class||'Recruit')+row('Level',p.level)+row('Experience',fmt(p.experience))+row('Paragon',p.paragon)+row('Renegade',p.renegade))+
   `<div class="buttons"><button class="btn primary full" data-roster="true">↳ BACK TO PRECINCT</button></div>`;
 }
 const data=state.precinct;
 if(!data)return heading('◈ PRECINCT / CONNECTION','Shared precinct','Contacting dispatch for a roster update…');
 const officers=(data.roster||[]).map(p=>`<div class="shop-item"><div class="shop-description">${officerDisplay(p)}</div><button class="btn small" data-officer="${escapeHTML(p.username)}">PROFILE</button></div>`).join('');
 const rankings=(data.leaderboard||[]).map((p,i)=>`<div class="shop-item"><div class="shop-description"><b>#${i+1} ${escapeHTML(p.username)}</b><span class="hint">${escapeHTML(p.officerName||'Recruit')} • Lv. ${fmt(p.level)} • ${fmt(p.experience)} XP</span></div><button class="btn small" data-officer="${escapeHTML(p.username)}">PROFILE</button></div>`).join('');
 return heading('◈ OPERATIONS / ONLINE PRECINCT','Shared online precinct','The city has other officers on duty. That should worry everybody.')+
  card('ON-DUTY ROSTER',officers||'<p class="hint">Nobody else has checked in recently.</p>')+
  card('EXPERIENCE LEADERBOARD',rankings||'<p class="hint">No ranked officers yet.</p>')+
  `<div class="buttons"><button class="btn cyan full" data-refresh-precinct="true">⟳ REFRESH PRECINCT</button></div>`;
}
async function loadPrecinct(){
 if(!state.auth?.authenticated)return;
 try{
  const response=await fetch('/api/precinct',{cache:'no-store'});
  const data=await response.json();
  if(!response.ok)throw Error(data.error||'Precinct unavailable');
  state.precinct=data; if(state.panel==='precinct')render();
 }catch(e){toast(e.message)}
}
async function accountSubmit(mode,username,password){
 if(state.busy)return;
 state.busy=true;
 try{
  const response=await fetch('/api/auth/'+mode,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({username,password})});
  const data=await response.json();
  if(!response.ok)throw Error(data.error||'Authentication failed');
  state.auth=data;
  const gameResponse=await fetch('/api/state',{cache:'no-store'});
  if(!gameResponse.ok)throw Error('Could not retrieve officer');
  state.game=await gameResponse.json();
  state.viewingOfficer=null;state.precinct=null;
  state.panel=state.game.phase==='hq'?'precinct':'patrol';
  render();
  if(state.panel==='precinct')await loadPrecinct();
 }catch(e){toast(e.message)}
 finally{state.busy=false}
}
async function logout(){
 if(state.busy)return;
 state.busy=true;
 try {
  const res=await fetch('/api/auth/logout',{method:'POST'});
  if(!res.ok)throw Error('Could not sign out');
  state.auth={authenticated:false,username:''};
  state.precinct=null;state.viewingOfficer=null;
  const game=await fetch('/api/state',{cache:'no-store'});
  if(!game.ok)throw Error('Could not load guest game');
  state.game=await game.json();state.panel='patrol';render();
 }catch(e){toast(e.message)}
 finally{state.busy=false}
}
async function showOfficer(username){
 try{
  const res=await fetch('/api/officers/'+encodeURIComponent(username),{cache:'no-store'});
  const data=await res.json();
  if(!res.ok)throw Error(data.error||'Officer not found');
  state.viewingOfficer=data;state.panel='precinct';render();
 }catch(e){toast(e.message)}
}

const pHasLSD = g => g.phase!=='setup' && g.player.lsdEncounters>0;
function render() {
 const g=state.game;
 if(!g) return;
 el('game-version').textContent=`• v${g.version || 'unknown'}`;
 let page;
 if((g.phase==='hq'||g.phase==='setup')&&state.panel==='account') page=authPage();
 else if((g.phase==='hq'||g.phase==='setup')&&state.panel==='precinct') page=precinctPage(g);
 else if(g.phase==='hq'&&g.shiftEnded&&!['account','precinct','record'].includes(state.panel)) page=shiftDonePage(g);
 else if(g.phase==='setup') page=setupPage();
 else if(g.phase==='redroom_result') page=redRoomResultPage(g);
 else if(g.phase!=='hq') page=patrolPage(g);
 else page=({patrol:patrolPage,hq:hqPage,property:propertyPage,nightlife:nightlifePage,doctor:doctorPage,record:recordPage}[state.panel] || patrolPage)(g);
 el('content').innerHTML=page;
 el('app').classList.toggle('lsd-active',pHasLSD(g));
 document.querySelectorAll('.nav-btn').forEach(b=>b.classList.toggle('active',b.dataset.panel===state.panel));
 document.querySelectorAll('.nav-btn').forEach(b=>b.disabled=g.phase!=='hq' && !(g.phase==='setup'&&['account','precinct','patrol'].includes(b.dataset.panel)));
 const p=g.player;
 el('officer-name').textContent=p.name || 'UNASSIGNED';
 el('officer-class').textContent=p.class ? `${p.class.toUpperCase()} // LEVEL ${p.level}` : 'CLASS NOT SELECTED';
 el('shift-status').textContent=g.phase==='setup'?'RECRUITMENT OPEN':g.phase==='gameover'?'CAMPAIGN FINISHED':g.phase==='combat'?'COMBAT ACTIVE':g.phase==='redroom_result'?'RED ROOM REPORT':g.shiftEnded?'OFF DUTY UNTIL 00:00 UTC':`${p.days} CALENDAR DAYS LEFT`;
 resetCountdown();
 if(g.phase==='setup') {el('vitals').innerHTML='';el('brief').innerHTML='<p class="hint">Choose a class to get started.</p>';return}
 const xpMax=([0,...[1000,4000,9000,18000,30000,60000,120000,240000,480000,960000,1920000,3840000,7680000,15360000]][p.level] || 15360000);
 el('vitals').innerHTML=progress('HEALTH',p.hp,Math.min(720,200+Math.max(0,p.level-1)*40),'#6addab')+progress('EXPERIENCE',p.experience,xpMax,'#ffc778')+(p.class==='psychic'?progress('PSYCHIC ENERGY',p.psychicEnergy??300,300,'#8a80ff'):p.class==='robot'?progress('CHARGED SHOT',p.classPower,4,'#57e8ef'):'')+
  `<div class="vital"><div class="vital-line"><span>KARMA</span><strong style="color:var(--pink)">${p.renegadeRank>=p.paragonRank?'RENEGADE':'PARAGON'}</strong></div><div class="bar"><div class="fill" style="width:${Math.min(100,Math.abs(p.renegadeRank-p.paragonRank)*10)}%;background:var(--pink)"></div></div></div>`;
 el('brief').innerHTML=row('CASH',`$${fmt(p.money)}`)+row('PATROLS',`${p.fightsLeft} / ${25}`)+row('DAYS',p.days)+row('INFECTIONS',p.infections)+row('HAND CUFFS',p.handcuffs||0)+row('ACTIVE EFFECT',activeEffect(p)?.[0]||'None');
}
function toast(msg){const box=el('toast');box.textContent=msg;box.classList.add('visible');clearTimeout(toast.timer);toast.timer=setTimeout(()=>box.classList.remove('visible'),4100)}
async function request(action,name='',officerClass=''){
 if(state.busy)return;
 state.busy=true;
 try{
  const response=await fetch('/api/action',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify({action,name,class:officerClass})});
  const data=await response.json();
  if(!response.ok)throw Error(data.error||'server error');
  state.game=data;
   // Buying gear should keep the player in the Property Room so updated
   // inventory is visible; only navigation actions leave the current panel.
   if(!(data.phase==='hq'&&(action.startsWith('buy_')||action.startsWith('use_'))))state.panel='patrol';
   render();
 }catch(err){toast(err.message || 'Connection lost')}
 finally{state.busy=false}
}
document.addEventListener('click',e=>{
 const cls=e.target.closest('[data-class]');if(cls){state.picked=cls.dataset.class;const name=el('new-name')?.value;render();if(el('new-name'))el('new-name').value=name||'';return}
 const nav=e.target.closest('[data-panel]');if(nav && ['hq','setup'].includes(state.game?.phase)){
  state.panel=nav.dataset.panel;state.viewingOfficer=null;render();
  if(state.panel==='precinct')loadPrecinct();
  return
 }
 const officer=e.target.closest('[data-officer]');if(officer){showOfficer(officer.dataset.officer);return}
 if(e.target.closest('[data-roster]')){state.viewingOfficer=null;render();return}
 if(e.target.closest('[data-refresh-precinct]')){loadPrecinct();return}
 if(e.target.closest('[data-logout]')){logout();return}
 const btn=e.target.closest('[data-action]');if(!btn)return;
 if(btn.dataset.action==='reset'&&!confirm('Replace this browser session save with a new game?'))return;
 request(btn.dataset.action,btn.dataset.action==='start'?(el('new-name')?.value||''):'',state.picked);
});
document.addEventListener('submit',e=>{
 const form=e.target.closest('[data-auth-form]');
 if(!form)return;
 e.preventDefault();
 accountSubmit(form.dataset.authForm,form.elements.username.value,form.elements.password.value);
});
setInterval(()=>{if(state.auth?.authenticated&&state.panel==='precinct'&&!state.viewingOfficer)loadPrecinct()},30000);
setInterval(()=>{resetCountdown();checkCalendar()},1000);
(async()=>{
 try{const [res,auth]=await Promise.all([
  fetch('/api/state',{cache:'no-store'}),fetch('/api/auth',{cache:'no-store'})
 ]);if(!res.ok||!auth.ok)throw Error('Cannot connect to OPP Wars server');state.game=await res.json();state.auth=await auth.json();render()}
 catch(e){el('content').innerHTML=heading('◈ OFFLINE','Connection failed','The server did not answer. Reload after starting the Go application.');toast(e.message)}
})();
