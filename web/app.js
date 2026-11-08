'use strict';

// One single source of truth: data.json, read and written by the local Go server.
// The browser never stores its own copy of the timer or settings in localStorage.
const $ = (selector) => document.querySelector(selector);
let state = null;
let updating = false;
let editing = false;
let editingRevision = null;
let offline = false;
let lastCompleted = null;
let statsSignature = null;
let selectedChartPeriod = null;
const els = {
  error: $('#error'), clock: $('#clock'), clockMinutes: $('#clock-minutes'), clockSeconds: $('#clock-seconds'), progress: $('#clock-progress'),
  timerState: $('#timer-state'), main: $('#main-action'), reset: $('#reset-action'),
  skip: $('#skip-action'), cycle: $('#cycle-label'), chart: $('#chart'), history: $('#history'),
  form: $('#settings-form'), accent: $('#accent'), accentLabel: $('#accent-label'), saved: $('#saved-label')
};
const names = {focus: 'Focus', break: 'Short break', long_break: 'Long break'};
const id = (name) => document.getElementById(name);
const pad = (n) => String(n).padStart(2, '0');
const minutes = (s) => Math.floor(s / 60);
const dateKey = (date) => `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}`;
const dateOffset = (days) => {const d = new Date();d.setHours(12,0,0,0);d.setDate(d.getDate()+days);return dateKey(d)};
function remaining(timer) {
  if (timer.status !== 'running' || !timer.ends_at) return timer.remaining_seconds;
  return Math.max(0, Math.min(timer.duration_seconds, Math.ceil((new Date(timer.ends_at).getTime()-Date.now()) / 1000)));
}
function showError(message) {els.error.hidden = !message;els.error.textContent = message || ''}
async function api(path, payload) {
  const response = await fetch(path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload),cache:'no-store'});
  const data = await response.json();
  if (!response.ok) throw new Error(data.error || `HTTP ${response.status}`);
  return data;
}
async function refresh() {
  if (updating || offline) return;
  try {
    const response = await fetch('/api/state', {cache:'no-store'});
    if (!response.ok || response.headers.get('X-Timo-App') !== '1') throw new Error('Local Timo server is unavailable.');
    const next = await response.json();
    if (lastCompleted !== null && next.history.length > lastCompleted) {
      const latest = next.history[next.history.length-1];
      if (latest.phase === 'focus') document.title = 'Focus complete · timo';
    }
    lastCompleted = next.history.length;
    state = next; showError(''); render();
  } catch(error) {showError(error.message)}
}
async function act(action) {
  if (updating || !state) return;
  updating = true;
  try {state = await api('/api/action',{action});lastCompleted=state.history.length;if(action==='start')document.title='timo';showError('');render()}
  catch(error) {showError(error.message);await refreshAfterError()}
  finally {updating=false;renderTimer()}
}
async function refreshAfterError() {
  try {const res=await fetch('/api/state',{cache:'no-store'});if(res.ok){state=await res.json();render()}}
  catch { /* The connection error is reported by the next poll. */ }
}
function renderTimer() {
  if (!state) return;
  const t = state.timer;
  const seconds = remaining(t);
  els.clockMinutes.textContent = pad(minutes(seconds));
  els.clockSeconds.textContent = pad(seconds % 60);
  els.clock.setAttribute('aria-label', `${minutes(seconds)} minutes ${seconds % 60} seconds remaining`);
  els.progress.style.width = `${100*(t.duration_seconds-seconds)/Math.max(1,t.duration_seconds)}%`;
  els.timerState.textContent = t.status === 'running' ? 'Running' : t.status === 'paused' ? 'Paused' : 'Ready';
  els.main.textContent = t.status === 'running' ? 'Pause' : t.status === 'paused' ? 'Resume' : 'Start';
  els.main.disabled = updating;
  document.querySelectorAll('[data-phase]').forEach(button=>{
    button.classList.toggle('selected',button.dataset.phase===t.phase);
    button.disabled=updating;
  });
  els.reset.disabled=updating;els.skip.disabled=updating;
  els.cycle.textContent = `Session ${(state.completed_focus % state.settings.long_break_every) + 1} / ${state.settings.long_break_every}`;
}
function filteredSessions(snapshot = state) {
  if(!snapshot) return [];
  const choice=id('range').value;
  if(choice==='all') return snapshot.history;
  const start=dateOffset(1-Number(choice));
  return snapshot.history.filter(x=>dateKey(new Date(x.completed_at))>=start);
}
function streak() {
  if (!state) return 0;
  const activeDays=new Set(state.history.filter(x=>x.phase==='focus').map(x=>dateKey(new Date(x.completed_at))));
  let offset=activeDays.has(dateOffset(0))?0:-1;
  let count=0;
  while(activeDays.has(dateOffset(offset))) {count++;offset--}
  return count;
}
function renderStats() {
  if (!state) return;
  const signature = `${state.history.length}:${state.history.at(-1)?.id ?? ''}:${id('range').value}`;
  statsSignature = signature;
  const sessions=filteredSessions();
  const focus=sessions.filter(x=>x.phase==='focus');
  const total=focus.reduce((sum,x)=>sum+x.duration_seconds,0);
  id('stat-minutes').textContent='';id('stat-minutes').append(document.createTextNode(String(Math.floor(total/60))));
  const m=document.createElement('span');m.textContent='m';id('stat-minutes').append(m);
  id('stat-sessions').textContent=String(focus.length);
  id('stat-streak').textContent='';id('stat-streak').append(document.createTextNode(String(streak())));
  const d=document.createElement('span');d.textContent='d';id('stat-streak').append(d);
  const values=new Map();
  for(const item of focus) {const key=dateKey(new Date(item.completed_at));values.set(key,(values.get(key)||0)+item.duration_seconds/60)}
  let start,days;
  const choice=id('range').value;
  if(choice==='all') {
    const keys=[...values.keys()].sort();
    const firstRecorded = new Date((keys[0] || dateOffset(-6))+'T12:00:00');
    const today = new Date(dateOffset(0)+'T12:00:00');
    days = Math.min(90, Math.max(7, Math.round((today-firstRecorded)/86400000)+1));
    start = dateOffset(-(days-1));
  } else {days=Number(choice);start=dateOffset(-(days-1))}
  const first=new Date(start+'T12:00:00');
  const daily=[];
  for(let n=0;n<days;n++){const dt=new Date(first);dt.setDate(first.getDate()+n);const key=dateKey(dt);daily.push({key,mins:values.get(key)||0})}
  // Dense ranges use seven-day buckets rather than forcing a wide chart.
  // On small screens, the 30-day view also switches to buckets for tap targets.
  const plotWidth=els.chart.parentElement.getBoundingClientRect().width || window.innerWidth;
  const bucketDays=days>30 || (days>7 && plotWidth<520) ? 7 : 1;
  const plot=[];
  for (let i=0; i<daily.length; i+=bucketDays) {
    const slice=daily.slice(i,i+bucketDays);
    plot.push({key:slice[0].key, end:slice[slice.length-1].key, mins:slice.reduce((sum,d)=>sum+d.mins,0)});
  }
  const max=Math.max(1,...plot.map(x=>x.mins));
  const chartDetail=id('chart-detail');
  const visiblePeriods=new Set(plot.map(x=>x.key));
  if (!visiblePeriods.has(selectedChartPeriod)) selectedChartPeriod=null;
  id('chart-subtitle').textContent = bucketDays===7 ? 'Focus minutes per 7-day period' : 'Focus minutes per day';
  els.chart.replaceChildren();
  for(const period of plot) {
    const slot=document.createElement('button');
    slot.type='button';slot.className='bar-slot';
    slot.dataset.day=period.key;
    slot.setAttribute('aria-pressed',String(selectedChartPeriod===period.key));
    const rounded=Math.round(period.mins);
    const dates=bucketDays===1 ? period.key : `${period.key} – ${period.end}`;
    slot.setAttribute('aria-label',`${dates}: ${rounded} focus minutes`);
    slot.title=`${dates}: ${rounded} min`;
    const bar=document.createElement('div');bar.className='bar'+(period.mins===0?' zero':'');
    bar.style.height=`${Math.max(1,100*period.mins/max)}%`;
    slot.addEventListener('click',()=>{
      selectedChartPeriod=period.key;
      els.chart.querySelectorAll('.bar-slot').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.day===selectedChartPeriod)));
      chartDetail.hidden=false;
      chartDetail.textContent=`${dates} · ${rounded} min of focus`;
    });
    slot.append(bar);els.chart.append(slot);
  }
  const selected=plot.find(x=>x.key===selectedChartPeriod);
  chartDetail.hidden=!selected;
  if (selected) chartDetail.textContent=`${selected.key}${bucketDays===1?'':` – ${selected.end}`} · ${Math.round(selected.mins)} min of focus`;
  id('chart-from').textContent=daily[0]?.key||'';
  id('chart-to').textContent=daily[daily.length-1]?.key||'';
  els.history.replaceChildren();
  if(sessions.length===0) {const empty=document.createElement('p');empty.className='empty';empty.textContent='No completed sessions yet. Your next focus session will appear here.';els.history.append(empty);return}
  for(const item of sessions.slice(-8).reverse()) {
    const row=document.createElement('div');row.className='history-row';
    for (const [cls,txt] of [['history-type',names[item.phase]],['history-date',new Date(item.completed_at).toLocaleString(undefined,{month:'short',day:'numeric',hour:'2-digit',minute:'2-digit'})],['history-duration',`${Math.round(item.duration_seconds/60)} min`]]) {
      const span=document.createElement('span');span.className=cls;span.textContent=txt;row.append(span);
    }
    els.history.append(row);
  }
}
function renderSettings() {
  if(!state || editing) return;
  const s=state.settings;
  id('focus-minutes').value=s.focus_minutes;id('break-minutes').value=s.break_minutes;
  id('long-break-minutes').value=s.long_break_minutes;id('long-break-every').value=s.long_break_every;
  els.accent.value=s.accent;els.accentLabel.textContent=s.accent;
  id('theme-mode').value=s.theme||'dark';id('grain-motion').checked=Boolean(s.grain_motion);
}
function render() {
  if(!state)return;
  renderTimer();
  if (!id('stats-panel').hidden && `${state.history.length}:${state.history.at(-1)?.id ?? ''}:${id('range').value}`!==statsSignature) renderStats();
  renderSettings();
  const cfg = editing ? {accent:els.accent.value,theme:id('theme-mode').value,grain_motion:id('grain-motion').checked} : state.settings;
  document.documentElement.style.setProperty('--accent',cfg.accent);
  document.documentElement.dataset.theme = cfg.theme || 'dark';
  document.documentElement.dataset.grainMotion = String(Boolean(cfg.grain_motion));
}
function selectTab(tab) {
  document.querySelectorAll('.nav-tab').forEach(button=>{
    const selected=button.dataset.tab===tab;
    button.classList.toggle('active',selected);
    if(selected)button.setAttribute('aria-current','page');else button.removeAttribute('aria-current');
  });
  document.querySelectorAll('.tab-panel').forEach(panel=>panel.hidden=panel.id!==`${tab}-panel`);
  if(tab==='stats' && state && `${state.history.length}:${state.history.at(-1)?.id ?? ''}:${id('range').value}`!==statsSignature)renderStats();
  if(tab==='settings')renderSettings();
}
document.querySelectorAll('.nav-tab').forEach(button=>button.addEventListener('click',()=>selectTab(button.dataset.tab)));
document.querySelectorAll('[data-phase]').forEach(button=>button.addEventListener('click',()=>act(button.dataset.phase)));
els.main.addEventListener('click',()=>act(state?.timer.status==='running'?'pause':state?.timer.status==='paused'?'resume':'start'));
els.reset.addEventListener('click',()=>act('reset'));
els.skip.addEventListener('click',()=>act('skip'));
id('range').addEventListener('change',()=>{statsSignature=null;renderStats()});
els.form.addEventListener('input',()=>{if (!editing) editingRevision=state?.settings_revision ?? 0;editing=true;els.saved.textContent='';els.accentLabel.textContent=els.accent.value.toUpperCase();document.documentElement.style.setProperty('--accent',els.accent.value);document.documentElement.dataset.theme=id('theme-mode').value;document.documentElement.dataset.grainMotion=String(id('grain-motion').checked)});
document.querySelectorAll('[data-color]').forEach(button=>button.addEventListener('click',()=>{els.accent.value=button.dataset.color;els.accent.dispatchEvent(new Event('input',{bubbles:true}))}));
els.form.addEventListener('submit',async(event)=>{
  event.preventDefault();if(!state||updating)return;
  const payload={focus_minutes:Number(id('focus-minutes').value),break_minutes:Number(id('break-minutes').value),long_break_minutes:Number(id('long-break-minutes').value),long_break_every:Number(id('long-break-every').value),accent:els.accent.value.toUpperCase(),theme:id('theme-mode').value,grain_motion:id('grain-motion').checked,expected_settings_revision:editingRevision ?? state.settings_revision};
  updating=true;
  try {state=await api('/api/settings',payload);editing=false;editingRevision=null;els.saved.textContent='Saved';showError('');render()}
  catch(error){if(error.message.includes('settings changed')){editing=false;editingRevision=null;await refreshAfterError()}showError(error.message)}
  finally{updating=false;renderTimer()}
});
window.addEventListener('keydown',event=>{
  if(event.code==='Space'&&!event.repeat&&document.activeElement===document.body&&state&&(!id('timer-panel').hidden)){
    event.preventDefault();els.main.click();
  }
});
if (location.protocol==='file:') {
  offline=true;
  showError('To start the timer, run timo web and open http://127.0.0.1:8765.');
  els.main.disabled=true;
  els.reset.disabled=true;
  els.skip.disabled=true;
  els.form.querySelector('button[type=submit]').disabled=true;
} else {
  refresh();setInterval(refresh,1500);setInterval(renderTimer,250);
}


function csvCell(value) {
  const raw=String(value ?? '');
  // Prevent Excel/LibreOffice from interpreting imported values as formulas.
  const safe=/^[\s]*[=+\-@]/.test(raw) ? `'${raw}` : raw;
  return `"${safe.replaceAll('"','""')}"`;
}
function sessionsCSV(sessions) {
  const columns=['id','phase','duration_seconds','duration_minutes','started_at','completed_at'];
  const rows=sessions.map(s=>[
    s.id,s.phase,s.duration_seconds,(s.duration_seconds/60).toFixed(2),s.started_at,s.completed_at
  ]);
  return '\uFEFF'+[columns,...rows].map(row=>row.map(csvCell).join(',')).join('\r\n')+'\r\n';
}
function downloadFile(filename, content, type) {
  const url=URL.createObjectURL(new Blob([content], {type}));
  const anchor=document.createElement('a');
  anchor.href=url;
  anchor.download=filename;
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  // Firefox needs the object URL to remain available until the download starts.
  window.setTimeout(()=>URL.revokeObjectURL(url),2000);
}
async function exportData(format) {
  if (updating || offline) return;
  const buttons=[id('export-csv'),id('export-json')];
  buttons.forEach(b=>b.disabled=true);
  try {
    const response=await fetch('/api/state',{cache:'no-store'});
    if (!response.ok || response.headers.get('X-Timo-App')!=='1') throw new Error('Could not read your data for export.');
    const snapshot=await response.json();
    const today=dateKey(new Date());
    if (format==='csv') {
      const period=id('range').value;
      downloadFile(`timo-sessions-${period==='all'?'all':period+'d'}-${today}.csv`,sessionsCSV(filteredSessions(snapshot)),'text/csv;charset=utf-8');
    } else {
      downloadFile(`timo-backup-${today}.json`,JSON.stringify(snapshot,null,2)+'\n','application/json;charset=utf-8');
    }
    showError('');
  } catch(error) {showError(error.message)}
  finally {buttons.forEach(b=>b.disabled=false)}
}
id('export-csv').addEventListener('click',()=>exportData('csv'));
id('export-json').addEventListener('click',()=>exportData('json'));
window.addEventListener('resize',()=>{if(state && !id('stats-panel').hidden) renderStats()});

document.addEventListener('visibilitychange',()=>{document.documentElement.dataset.pageHidden=String(document.hidden)});
