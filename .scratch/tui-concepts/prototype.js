/* Three throwaway TUI structures on one route, switchable with ?variant=A|B|C.
   All data is fictional. This file never invokes SSH or reads product state. */
'use strict';
const concepts = {
  A: {name:'Grouped panels', description:'Separate identity, status, and actions into bordered areas. Wide screens put related facts next to each other; narrow screens stack the same panels.'},
  B: {name:'Guided pages', description:'One plain-language question or answer at a time. Lead with the useful conclusion, show one next step, and move supporting facts below it.'},
  C: {name:'Compact tables', description:'Stable columns, short labels, and explicit values. Compare records quickly, with details and commands arranged as a compact reference sheet.'}
};
const pages = [
  ['destinations','Browse','Destinations','list'],
  ['keys','Browse','Keys','list'],
  ['registrations','Browse','Registrations','list'],
  ['tasks','Browse','Tasks','list'],
  ['destination','Details','Destination overview','detail'],
  ['key','Details','Key overview','detail'],
  ['registration','Details','Registration overview','detail'],
  ['task','Details','Saved task overview','detail'],
  ['multiple','Details','Several keys · mixed outcomes','scenario'],
  ['duplicates','Details','Duplicate key names','scenario'],
  ['config-only','Details','Configuration-only recovery','scenario'],
  ['picker','Prepare','Choose a key','picker'],
  ['destination-picker','Prepare','Choose a destination','picker'],
  ['registration-picker','Prepare','Choose a registration','picker'],
  ['recovery-picker','Prepare','Choose saved work','picker'],
  ['create','Prepare','Create a key','form'],
  ['import','Prepare','Import a key','form'],
  ['add','Prepare','Add destination · name','form'],
  ['access','Prepare','Add destination · access','form'],
  ['setup','Prepare','Add destination · options','form'],
  ['register','Prepare','Register a public key','form'],
  ['unregister','Prepare','Unregister a public key','form'],
  ['delete','Prepare','Delete local key files','form'],
  ['resume','Prepare','Continue saved work','form'],
  ['validation','Prepare','Form with an error','form'],
  ['review','Decide','Review changes','review'],
  ['discard','Decide','Discard draft?','discard'],
  ['handoff','Decide','Terminal handoff','handoff'],
  ['success','Outcomes','Registration completed','result'],
  ['manual','Outcomes','Manual instructions ready','result'],
  ['partial','Outcomes','Partially completed','result'],
  ['include','Outcomes','SSH Include unfinished','scenario'],
  ['unknown','Outcomes','Registration outcome unknown','result'],
  ['failed','Outcomes','Connection check failed','result'],
  ['cancelled','Outcomes','Cancelled before execution','result'],
  ['help','Read','Contextual help','document'],
  ['history','Read','Destination history','document'],
  ['empty-history','Read','History · no events','scenario'],
  ['discovery','Read','Found on this computer','document'],
  ['technical','Read','Technical details','document'],
  ['long-values','Read','Technical · long values','scenario'],
  ['instructions','Read','Session instructions','document'],
  ['include-instructions','Read','SSH Include instructions','document'],
  ['session-output','Read','Session output · SSH error','document'],
  ['empty','Edge states','No destinations yet','edge'],
  ['no-tasks','Edge states','No saved tasks','edge'],
  ['search','Edge states','Search · no matches','edge'],
  ['stale','Edge states','Refresh failed','edge'],
  ['blocked','Edge states','Missing key reference','edge'],
  ['small','Edge states','Terminal below minimum','edge']
].map(([id,group,title,type])=>({id,group,title,type}));
const query = new URLSearchParams(location.search);
const state = {
  variant:concepts[query.get('variant')] ? query.get('variant') : 'A',
  page:pages.some(p=>p.id===query.get('page')) ? query.get('page') : 'destination',
  size:['80x24','60x18','32x12'].includes(query.get('size')) ? query.get('size') : '80x24',
  color:query.get('color') !== 'off', compare:query.get('compare') === 'all',
  offsets:{A:0,B:0,C:0}, selected:0, field:0, trail:[], notice:''
};
const $ = id=>document.getElementById(id);
const cut = (text,width)=>text.length>width ? text.slice(0,Math.max(0,width-1))+'…' : text;
const pad = (text,width)=>cut(text,width).padEnd(width);
function wrap(text,width) {
  if(!text) return [''];
  const result=[];
  for(const paragraph of String(text).split('\n')) {
    let line='';
    for(let word of paragraph.split(' ')) {
      if(line && line.length+word.length+1>width){ result.push(line);line=''; }
      while(word.length>width){ if(line){result.push(line);line='';} result.push(word.slice(0,width));word=word.slice(width); }
      if(word)line+=(line?' ':'')+word;
    }
    result.push(line);
  }
  return result;
}
const prose=(text,width)=>wrap(text,width);
function box(title,lines,width) {
  const label=` ${title} `;
  return ['┌'+cut(label,width-2).padEnd(width-2,'─')+'┐',
    ...lines.flatMap(line=>wrap(line,width-4)).map(line=>'│ '+pad(line,width-4)+' │'),
    '└'+'─'.repeat(width-2)+'┘'];
}
function columns(left,right,width) {
  const lw=left[0].length,rw=width-lw-2;
  return Array.from({length:Math.max(left.length,right.length)},(_,i)=>pad(left[i]||'',lw)+'  '+pad(right[i]||'',rw));
}
function pairs(items,width) {
  const keyWidth=Math.min(16,Math.floor(width*.36));
  return items.flatMap(([key,value])=>{
    if(width<38)return [key+':',...wrap(value,width-2).map(line=>'  '+line)];
    const wrapped=wrap(value,width-keyWidth-2);
    return wrapped.map((line,i)=>pad(i?'':key,keyWidth)+'  '+line);
  });
}
function table(headers,rows,width) {
  if(width<50)return rows.flatMap((row,i)=>[
    `${i===state.selected?'>':' '} ${row[0]}`,
    ...row.slice(1).flatMap((value,j)=>wrap(`${headers[j+1]}: ${value}`,width-2).map(line=>'  '+line)), ''
  ]);
  const widths=headers.length===3?[Math.floor(width*.31),Math.floor(width*.38),0]:[Math.floor(width*.37),0];
  widths[widths.length-1]=width-widths.slice(0,-1).reduce((a,b)=>a+b,0)-(headers.length-1)*2-2;
  const rowText=row=>row.map((value,i)=>pad(value,widths[i])).join('  ');
  return ['  '+rowText(headers),'  '+'─'.repeat(width-2),...rows.map((row,i)=>(i===state.selected?'> ':'  ')+rowText(row))];
}
const lists={
  destinations:{heading:'Destinations',question:'Which destination do you want to manage?',headers:['Name','Login','Saved work'],rows:[
    ['lab-box','sam@192.0.2.24:22','Unfinished'],
    ['build-runner','ci@build.example.test:22','None'],
    ['storage','sam@192.0.2.48:2222','None']]},
  keys:{heading:'Keys',question:'Which key do you want to use?',headers:['Key','Saved paths','Reference'],rows:[
    ['work-laptop','Private + public','…7mQ2'],
    ['backup-key','Public only','…4pR8'],
    ['old-desktop','Private + public','…9kT1']]},
  registrations:{heading:'Registrations',question:'Which destination and key pair?',headers:['Destination / key','Registration','Connection'],rows:[
    ['lab-box / work-laptop','Unknown outcome','Not checked'],
    ['build-runner / work-laptop','Recorded complete','Passed previously'],
    ['storage / backup-key','Recorded removed','Not checked']]},
  tasks:{heading:'Saved tasks',question:'Which unfinished operation needs your attention?',headers:['Task','Why it stopped','Next step'],rows:[
    ['Register at lab-box','Result not captured','Review saved work'],
    ['Finish SSH config','Include not confirmed','Read instructions']]},
  picker:{heading:'Choose a key',question:'Which public key should be registered?',headers:['Key','Saved paths','Reference'],rows:[
    ['work-laptop','Private + public','…7mQ2'],
    ['backup-key','Public only','…4pR8']]},
  'destination-picker':{heading:'Choose a destination',question:'Where should this key be registered?',headers:['Destination','Login','OS'],rows:[
    ['lab-box','sam@192.0.2.24:22','Linux'],['build-runner','ci@build.example.test:22','Linux']]},
  'registration-picker':{heading:'Choose a registration',question:'Which public key should be removed from the destination?',headers:['Destination / key','Registration','Connection'],rows:[
    ['lab-box / work-laptop','Unknown outcome','Not checked'],['storage / backup-key','Recorded complete','Passed previously']]},
  'recovery-picker':{heading:'Choose saved work',question:'Which saved operation do you want to review?',headers:['Operation','Destination','Why unfinished'],rows:[
    ['Register public key','lab-box','Result not captured'],['Finish SSH config','storage','Include not confirmed']]}
};
function listActions(id){
  return {destinations:'a Add destination',keys:'c Create key · i Import key',
    registrations:'r Register · u Unregister',tasks:'Enter review saved task'}[id] || 'Enter choose · Esc cancel';
}
function listPage(id,variant,width) {
  const data=lists[id],rows=data.rows;
  let context=[];
  if(id==='picker')context=['Register at lab-box · sam@192.0.2.24:22'];
  if(id==='destination-picker')context=['Register key: work-laptop · ref …7mQ2'];
  if(id==='registration-picker')context=['Unregister removes the remote public key only.'];
  if(id==='recovery-picker')context=['Selection opens a draft; nothing runs yet.'];
  if(variant==='C')return [...prose(data.heading,width),...context.flatMap(x=>prose(x,width)),'',...table(data.headers,rows,width),'',
    ...prose(id==='tasks'?'Enter review task · no automatic retry':listActions(id)+' · / search',width)];
  if(variant==='B')return [...prose(data.question,width),'',...context.flatMap(x=>prose(x,width)),
    ...rows.flatMap((row,i)=>[
      ...prose(`${state.selected===i?'>':' '} ${i+1}. ${row[0]}`,width),
      ...prose('   '+row.slice(1).join(' · '),width),'']),
    ...prose('Enter chooses the highlighted item.',width),
    ...prose(id==='tasks'?'These are saved operations, not live server health.':listActions(id)+' · ? help',width)];
  return [...context.flatMap(x=>prose(x,width)),...(context.length?['']:[]),
    ...box(data.heading+' · '+rows.length+' items',rows.flatMap((row,i)=>[
      `${state.selected===i?'>':' '} ${row[0]}`,`  ${row.slice(1).join(' · ')}`,i<rows.length-1?'':''
    ]),width),'',...prose('Enter open selection · / search',width),
    ...prose(id==='tasks'?'Saved work only; not a live server check.':
      listActions(id)+(id.endsWith('picker')?'':' · Tab switch category'),width)];
}
const details={
  destination:{name:'lab-box',kind:'Destination',subtitle:'sam@192.0.2.24:22 · Linux',
    headline:'Registration needs review',
    explanation:'The previous registration did not record a final result. The key may or may not be on this destination.',
    facts:[['Key','work-laptop · ref …7mQ2'],['Registration','Outcome unknown'],['Connection','Not checked'],['SSH alias','No binding recorded']],
    next:'s Review saved work',reason:'See where it stopped before deciding whether to continue.',
    other:['r Register another key','u Unregister a public key'],technical:'destination-01'},
  key:{name:'work-laptop',kind:'Key',subtitle:'Ed25519 · ref …7mQ2',
    headline:'Private and public paths are saved',
    explanation:'Saved paths do not establish that the files are currently usable or where the public key is registered.',
    facts:[['Private path','~/.ssh/work-laptop'],['Public path','~/.ssh/work-laptop.pub'],['lab-box','Registration unknown'],['Connection','Not checked']],
    next:'r Choose a destination',reason:'Select where to register this public key.',
    other:['d Delete local key files','h View recorded history'],technical:'key-01'},
  registration:{name:'lab-box / work-laptop',kind:'Registration',subtitle:'sam@192.0.2.24:22 · key ref …7mQ2',
    headline:'Registration outcome is unknown',
    explanation:'The saved operation ended without a confirmed registration result. No connection check is recorded.',
    facts:[['Destination','lab-box'],['Key','work-laptop · ref …7mQ2'],['Registration','Outcome unknown'],['Connection','Not checked']],
    next:'s Review saved work',reason:'Inspect the saved checkpoint before continuing.',
    other:['u Unregister this public key','h View recorded history'],technical:'registration-01'},
  task:{name:'Register at lab-box',kind:'Saved task',subtitle:'work-laptop → sam@192.0.2.24:22',
    headline:'The final result was not captured',
    explanation:'Registration started, but no final result was saved. Remote changes may already exist.',
    facts:[['Operation','Register public key'],['Registration','Outcome unknown'],['Connection','Not checked'],['Continuation','Review available']],
    next:'s Review continuation',reason:'Check the exact target and checkpoint; terminal approval is still required.',
    other:['v Inspect checkpoint','h View recorded history'],technical:'task-01'}
};
function detailPage(id,variant,width) {
  const data=details[id];
  if(width<50){
    const quickFacts=id==='key'?['Paths: private + public saved','File usability: not confirmed']:['Registration: unknown','Connection: not checked'];
    if(variant==='A')return [...box(data.name,[...quickFacts,'> '+data.next],width),'',
      ...prose(data.explanation,width),'',...prose(data.subtitle,width),'',
      ...pairs(data.facts,width),'',...data.other.flatMap(x=>prose(x,width))];
    if(variant==='B')return [...prose(data.name,width),...prose(data.headline,width),
      ...prose('> '+data.next,width),...prose(quickFacts[1],width),'',
      ...prose(data.explanation,width),'',...prose(data.subtitle,width),'',...pairs(data.facts,width),
      '',...data.other.flatMap(x=>prose(x,width))];
    return [...prose(data.name,width),...quickFacts.flatMap(x=>prose(x,width)),
      ...prose('Next: '+data.next,width),'',...prose(data.subtitle,width),'',
      ...pairs(data.facts,width),'',...prose('Why: '+data.explanation,width),
      '',...data.other.flatMap(x=>prose(x,width))];
  }
  if(variant==='B')return [
    ...prose(data.name,width),...prose(data.subtitle,width),'',
    ...prose(data.headline,width),...prose(data.explanation,width),'',
    ...prose('> '+data.next,width),...prose(data.reason,width),'',
    'What is known',...pairs(data.facts,width),'',
    'Other options',...data.other.flatMap(x=>prose(x,width)),'',
    ...prose('v full details · h history',width),
    ...prose('Status describes saved records, not live SSH health.',width)];
  if(variant==='C')return [
    ...prose(`${data.kind} / ${data.name}`,width),...prose(data.subtitle,width),'',
    ...pairs(data.facts,width),'',
    ...prose('Why: '+data.explanation,width),'',
    ...prose('NEXT  '+data.next,width),...prose(data.reason,width),'',
    ...pairs(data.other.map(item=>[item.slice(0,1),item.slice(2)]).concat([['v','Full details'],['h','History']]),width),
    '',...prose('Snapshot: saved records; no live SSH check.',width)];
  const identity=box(data.kind,[data.name,data.subtitle],width);
  const status=width>=70 ? columns(
    box('Status',data.facts.map(([k,v])=>`${k}: ${v}`),Math.floor((width-2)*.54)),
    box('Next step',[data.next,'',data.reason],width-Math.floor((width-2)*.54)-2),width)
    : [...box('Status',data.facts.map(([k,v])=>`${k}: ${v}`),width),'',...box('Next step',[data.next,data.reason],width)];
  return [...identity,...prose(data.headline,width),
    ...(width<70?[...prose('> '+data.next,width)]:[]),'',...status,'',
    ...prose(data.explanation,width),'',
    ...prose(data.other.join(' · '),width),...prose('v full details · h history',width),
    ...prose('Saved records only; no live SSH check.',width)];
}
const forms={
  create:{title:'Create an Ed25519 key',question:'What should this key be called?',context:'A new key pair will be created on this computer.',fields:[['Key name','work-laptop'],['Comment','sam@example.test']],focus:0,note:'Passphrase entry happens privately in the terminal.',next:'review'},
  import:{title:'Import a local key',question:'Where is the existing key file?',context:'Import records this key. It does not register it on a destination.',fields:[['Key name','backup-key'],['Private key path','~/.ssh/id_ed25519']],focus:1,note:'Use a private key path or choose public-only import.',next:'review'},
  add:{title:'Add a destination',question:'What destination are you adding?',context:'Destination setup · 1 of 3',fields:[['Name','lab-box'],['Host or IP','192.0.2.24'],['Login user','sam'],['Port','22'],['Operating system','Linux']],focus:1,note:'This step saves a target; it does not connect.',next:'access'},
  access:{title:'Choose initial access',question:'How will you access lab-box?',context:'Destination setup · 2 of 3 · sam@192.0.2.24:22',fields:[['Authentication','Existing key'],['Authentication key','~/.ssh/bootstrap_key']],focus:0,note:'Passwords and passphrases are entered only in the terminal.',next:'setup'},
  setup:{title:'Choose setup options',question:'What should setup prepare?',context:'Destination setup · 3 of 3 · lab-box',fields:[['Public key','work-laptop · ref …7mQ2'],['Register public key','Yes'],['SSH configuration','Do not create alias']],focus:1,note:'You will review the exact effects before execution.',next:'review'},
  register:{title:'Register a public key',question:'How should this public key be registered?',context:'work-laptop → lab-box · sam@192.0.2.24:22',fields:[['Authentication','Existing key'],['Authentication key','~/.ssh/bootstrap_key'],['SSH configuration','Do not create alias']],focus:0,note:'Only the public key is sent. Secrets stay in the terminal.',next:'review'},
  unregister:{title:'Unregister a public key',question:'How will you confirm alternate access?',context:'Remove work-laptop from lab-box · local key files stay.',fields:[['Alternate access','Existing key'],['Alternate key','~/.ssh/backup_admin']],focus:1,note:'Fresh alternate authentication is required. Final checks happen in the terminal.',next:'review'},
  delete:{title:'Delete local key files',question:'Which local files would be removed?',context:'Key: old-desktop · this does not unregister any remote public key.',fields:[['Private file','~/.ssh/old-desktop'],['Public file','~/.ssh/old-desktop.pub']],focus:0,note:'Read-only file summary. Exact deletion approval happens in the terminal.',next:'review'},
  resume:{title:'Continue saved registration',question:'Are these still the intended target and method?',context:'Saved work: register work-laptop at lab-box.',fields:[['Destination','lab-box · sam@192.0.2.24:22'],['Key','work-laptop · ref …7mQ2'],['Authentication','Existing key']],focus:2,note:'The previous result is unknown. Continuation revalidates the saved target before any retry.',next:'review'},
  validation:{title:'Add a destination',question:'Which login user should SSH use?',context:'Destination setup · 1 of 3',fields:[['Name','lab-box'],['Host or IP','192.0.2.24'],['Login user','[empty]'],['Port','22']],focus:2,error:'Enter the SSH login user, for example sam.',note:'Your other fields have been kept.',next:'access'}
};
function formPage(id,variant,width) {
  const data=forms[id],focus=(data.focus+state.field)%data.fields.length;
  if(width<50){
    const [key,value]=data.fields[focus],entry=['> '+key,...prose(value+(id==='delete'?'':'_'),width-4)];
    const error=data.error?prose('Error: '+data.error,width):[];
    const rest=['',...prose(data.context,width),'',...pairs(data.fields.filter((_,i)=>i!==focus),width),
      '',...prose(data.note,width),'','[Review]  Cancel'];
    if(variant==='A')return [...box(data.title,entry,width),...error,...rest];
    if(variant==='B')return [...prose(data.question,width),'',...entry,...error,
      ...prose('Enter next · Tab another field',width),...rest];
    return [cut(data.title,width),...entry,...error,'Enter review · Tab another field',...rest];
  }
  const fields=data.fields.flatMap(([key,value],i)=>{
    const suffix=i===focus && id!=='delete'?'_':'';
    return variant==='C' ? pairs([[(i===focus?'> ':'  ')+key,value+suffix]],width)
      : [...prose(`${i===focus?'>':' '} ${key}`,width),...prose('  '+value+suffix,width),'',
        ...(data.error&&i===data.focus?prose('Error: '+data.error,width):[])];
  });
  const end=[...prose(data.note,width),'',...prose('[Continue to review]  Cancel',width)];
  if(['add','access'].includes(id))end[end.length-1]='[Next step]  Cancel';
  if(variant==='B'){
    const [key,value]=data.fields[focus];
    return [...prose(data.context,width),'',...prose(data.question,width),'',
      ...prose('> '+key,width),...prose('  '+value+(id==='delete'?'':'_'),width),
      ...(data.error?['',...prose('Error: '+data.error,width)]:[]),'',
      ...prose(data.note,width),'',...prose('Already in this draft',width),
      ...pairs(data.fields.filter((_,i)=>i!==focus),width),'',
      ...prose(id==='delete'?'[Review deletion]':'[Next]  Back  Cancel',width)];
  }
  if(variant==='C')return [...prose(data.title,width),...prose(data.context,width),'',...fields,
    ...(data.error?['',...prose('Error: '+data.error,width)]:[]),'',...end];
  return [...prose(data.context,width),'',...box(data.title,fields,width),'',...end];
}
const outcomes={
  success:{headline:'Registration recorded complete',summary:'The public key was registered at lab-box.',facts:[['Registration','Recorded complete'],['Connection check','Passed during this run'],['SSH alias','Not requested']],next:'Enter Open registration',more:'o Session output · h History'},
  manual:{headline:'Instructions ready — registration unconfirmed',summary:'No automated registration ran. Complete the instructions in your own terminal.',facts:[['Remote key','Not confirmed'],['Connection','Not checked'],['Instructions','Available for this session']],next:'o Read instructions',more:'Enter Open destination'},
  partial:{headline:'Key registered; connection check failed',summary:'Registration was recorded, but this key did not pass the SSH connection check.',facts:[['Registration','Recorded complete'],['Connection','Failed in this run'],['Error','Permission denied (publickey)']],next:'o Read connection error',more:'Enter Open registration · h History'},
  unknown:{headline:'Registration outcome is unknown',summary:'The session ended before a final result was saved. The destination may already have changed.',facts:[['Registration','Outcome unknown'],['Connection','Not checked'],['Saved work','Available for review']],next:'s Review saved work',more:'o Session output · Enter Open destination'},
  failed:{headline:'The connection check failed',summary:'SSH could not authenticate with work-laptop. This does not prove whether the public key is registered.',facts:[['Registration','Previously recorded complete'],['Connection','Failed in this run'],['Error','Permission denied (publickey)']],next:'o Read connection error',more:'Enter Open registration · h History'},
  cancelled:{headline:'Cancelled before execution',summary:'You returned before the SSH operation started. This attempt made no remote changes.',facts:[['SSH operation','Not started'],['Registration','Unchanged by this attempt'],['Draft','Closed']],next:'Enter Back to destinations',more:'? Help'}
};
function resultPage(id,variant,width) {
  const data=outcomes[id],target='lab-box · work-laptop · sam@192.0.2.24:22';
  if(width<50){
    const rest=['',...prose(target,width),'',...pairs(data.facts,width),'',...prose(data.summary,width),'',...prose(data.more,width)];
    if(variant==='A')return [...box('lab-box / work-laptop',[data.headline,'> '+data.next],width),...rest];
    if(variant==='B')return [...prose(data.headline,width),...prose('> '+data.next,width),...rest];
    return ['lab-box / work-laptop',...prose(data.headline,width),...prose('Next: '+data.next,width),...rest];
  }
  if(variant==='B')return [...prose(data.headline,width),'',...prose(data.summary,width),'',
    'NEXT',...prose('> '+data.next,width),'',...prose(target,width),'',
    'What this run recorded',...pairs(data.facts,width),'',...prose(data.more,width)];
  if(variant==='C')return [...prose('Result / '+data.headline,width),...prose(target,width),'',
    ...pairs(data.facts,width),'',...prose(data.summary,width),'',
    ...prose('NEXT  '+data.next,width),...prose(data.more,width)];
  return [...box('Result',[data.headline,data.summary],width),'',
    ...box('Target',[target],width),'',...box('This run',data.facts.map(([k,v])=>k+': '+v),width),'',
    ...prose('> '+data.next,width),...prose(data.more,width)];
}
const documents={
  help:{title:'Help / destination overview',lead:'Read the status, then choose the next step.',sections:[
    ['On this page',['s  Review unfinished registration','r  Register another public key','u  Unregister a public key; keep local files','v  Open full technical details','h  Read this destination’s recorded history']],
    ['Reading status',['Registration and connection checks are separate.','Unknown means a final outcome was not captured.','Not checked means no connection check was recorded.','These records do not establish live server health.']],
    ['Navigation',['Up/Down scroll · PgUp/PgDn move a page','Esc return · F5 refresh saved records']]]},
  history:{title:'History / lab-box',lead:'Recorded events for this destination; no live SSH check.',sections:[
    ['Most recent',['2026-11-08 10:42  Registration outcome unknown','Key: work-laptop · ref …7mQ2','The session ended before a final result was saved.']],
    ['Earlier',['2026-11-08 10:40  Registration started','Authentication: existing key','2026-11-07 09:12  Destination added']],
    ['Scope',['Only events linked to destination-01 are shown.','Esc returns to the destination overview.']]]},
  discovery:{title:'Found on this computer',lead:'Found files are not automatically managed or imported.',sections:[
    ['Not managed by KeyFlow',['~/.ssh/id_ed25519.pub','Public key found; no KeyFlow import record.','~/.ssh/config · Host example-lab','SSH entry found; not a managed destination.']],
    ['What to do',['Return to Keys and choose Import to review a key.','Return to Destinations and choose Add to save a target.','Viewing this page does not import or modify anything.']],
    ['Discovery diagnostics',['Placeholder report: one unmanaged public key found.','Placeholder report: one unmanaged SSH entry found.','No SSH connection was attempted.']]]},
  technical:{title:'Technical details / lab-box',lead:'Full saved references. Read-only.',sections:[
    ['Destination',['ID: destination-01','Alias: lab-box','User: sam','Host: 192.0.2.24','Port: 22','OS: linux']],
    ['Registration',['ID: registration-01','Key ID: key-01','Key name: work-laptop','Fingerprint: SHA256:EXAMPLEPLACEHOLDER000000007mQ2','Registration outcome: unknown','Connection evidence: not_checked']],
    ['Checkpoint',['Task ID: task-01','Operation: register','Phase: awaiting saved final outcome','Remote effects may already exist.']],
    ['Configuration',['SSH alias binding: none recorded','Private path: ~/.ssh/work-laptop','Public path: ~/.ssh/work-laptop.pub']]]},
  instructions:{title:'Session instructions / manual registration',lead:'Placeholder commands for illustration; this prototype runs nothing.',sections:[
    ['Before starting',['Check that lab-box and work-laptop are the intended pair.','Use your trusted access method to reach the destination.']],
    ['Public key to install',['ssh-ed25519 AAAA...PLACEHOLDER... sam@example.test','Append the intended public key to the target authorized_keys.','Do not copy or upload your private key.']],
    ['After installation',['Test access using your normal trusted SSH workflow.','A prepared instruction page does not confirm registration.']],
    ['Session output',['No automated SSH registration was attempted.','This document is read-only. Esc returns to the result.']]]}
  ,
  'include-instructions':{title:'Session instructions / SSH Include',lead:'Illustrative configuration evidence; no commands run here.',sections:[
    ['What was saved',['Alias file: ~/.ssh/ssh-keyflow/config','Alias: lab-box','Registration and connection results remain independent.']],
    ['What is unfinished',['The application did not confirm an active Include for this file.','Review the full session output and your SSH configuration.','Do not assume that the saved alias is effective yet.']],
    ['Next',['Use the exact instructions from the real operation after reviewing their paths.','This prototype does not invent or execute a repair command.']]]},
  'session-output':{title:'Session output / connection check',lead:'Illustrative error output for this run. Read-only.',sections:[
    ['Target',['lab-box · sam@192.0.2.24:22','Key: work-laptop · ref …7mQ2']],
    ['SSH output',['sam@192.0.2.24: Permission denied (publickey).']],
    ['Interpretation',['This connection attempt did not authenticate.','This message alone does not establish whether the public key is registered.','No automatic retry is offered by reading this document.']]]}
};
function scenarioPage(id,variant,width){
  const data={
    multiple:{title:'lab-box / several keys',lead:'Each key has its own registration and connection record.',headers:['Key / reference','Registration','Connection'],rows:[
      ['work-laptop / …7mQ2','Outcome unknown','Not checked'],
      ['backup-key / …4pR8','Recorded complete','Passed previously'],
      ['old-desktop / …9kT1','Recorded removed','Not checked']],next:'Enter Open the highlighted pair'},
    duplicates:{title:'Choose a key / duplicate names',lead:'Same name; distinguish by reference and public path.',headers:['Key / reference','Saved public path','Use'],rows:[
      ['work-laptop / …7mQ2','~/.ssh/work-laptop.pub','Primary'],
      ['work-laptop / …4pR8','~/.ssh/archive/laptop.pub','Backup']],next:'Enter Review the exact key'},
    'config-only':{title:'SSH configuration needs manual review',lead:'No supported continuation exists for this saved configuration-only work.',facts:[
      ['Destination','lab-box'],['Alias file','Recorded saved'],['SSH Include','Not confirmed'],['Continue','Unavailable']],next:'v Inspect configuration evidence',
      tail:'Review the saved paths and Include evidence. Do not repeat registration to repair a configuration-only checkpoint.'},
    include:{title:'Key registered; SSH Include unfinished',lead:'The alias file was saved, but KeyFlow did not confirm that SSH includes it.',facts:[
      ['Registration','Recorded complete'],['Connection','Passed during this run'],['Alias file','Recorded saved'],['SSH Include','Not confirmed']],next:'o Read configuration instructions',
      tail:'The alias may not work yet. Saved alias content and an active SSH Include are separate facts.'},
    'empty-history':{title:'History / backup-key',lead:'No recorded events for this key.',facts:[
      ['Scope','key-02 / backup-key'],['Events','0']],next:'Esc Return to key overview',
      tail:'An empty history does not prove that this public key was never registered elsewhere.'},
    'long-values':{title:'Technical / full references',lead:'Full values wrap and remain scrollable. Nothing is shortened in this document.',facts:[
      ['Fingerprint','SHA256:PLACEHOLDER0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ'],
      ['Private path','C:/Users/placeholder-user/.ssh/team-environments/engineering-development/region-one/rotation-2026/work-laptop_ed25519'],
      ['Public path','C:/Users/placeholder-user/.ssh/team-environments/engineering-development/region-one/rotation-2026/work-laptop_ed25519.pub'],
      ['Destination ID','destination-placeholder-00000000-1111-2222-3333-444444444444']],next:'Esc Return to overview',
      tail:'References identify the saved target; they are not evidence of current access.'}
  }[id];
  if(data.rows){
    if(variant==='C')return [...prose(data.title,width),...prose(data.lead,width),'',
      ...table(data.headers,data.rows,width),'',...prose(data.next,width)];
    if(variant==='A')return [...prose(data.title,width),...prose(data.lead,width),'',
      ...data.rows.flatMap((row,i)=>[...box((i===state.selected?'> ':'')+row[0],row.slice(1).map((value,j)=>data.headers[j+1]+': '+value),width),'']),
      ...prose(data.next,width)];
    return [...prose(data.title,width),'',...prose('Which key do you want to inspect?',width),'',
      ...data.rows.flatMap((row,i)=>[...prose((i===state.selected?'> ':'  ')+row[0],width),
        ...row.slice(1).flatMap((value,j)=>prose(data.headers[j+1]+': '+value,width)),'']),
      ...prose(data.lead,width),...prose(data.next,width)];
  }
  if(variant==='A')return [...box(data.title,[data.lead],width),'',...prose('> '+data.next,width),'',
    ...box('Saved facts',data.facts.map(([key,value])=>key+': '+value),width),'',...prose(data.tail,width)];
  if(variant==='B')return [...prose(data.title,width),'',...prose(data.lead,width),'',
    ...prose('> '+data.next,width),'',...pairs(data.facts,width),'',...prose(data.tail,width)];
  return [...prose(data.title,width),'',...pairs(data.facts,width),'',
    ...prose(data.lead,width),...prose('Next: '+data.next,width),'',...prose(data.tail,width)];
}
function documentPage(id,variant,width){
  let data=documents[id];
  if(id==='session-output'){
    const origin=state.trail[state.trail.length-1];
    if(origin==='success')data={title:'Session output / completed registration',lead:'Illustrative output for this completed run. Read-only.',sections:[
      ['Target',['lab-box · sam@192.0.2.24:22','Key: work-laptop · ref …7mQ2']],
      ['Recorded result',['Registration recorded complete.','Connection check passed during this run.','No SSH alias was requested.']],
      ['Scope',['These results describe this run, not present server health.']]]};
    if(origin==='unknown')data={title:'Session output / unknown outcome',lead:'Illustrative session evidence. Read-only.',sections:[
      ['Target',['lab-box · sam@192.0.2.24:22','Key: work-laptop · ref …7mQ2']],
      ['Available evidence',['Registration started.','The session ended before a final result was saved.','No connection check is recorded.']],
      ['Scope',['The destination may already have changed.','Review saved work before deciding whether to continue.']]]};
  }
  if(variant==='A')return [...prose(data.title,width),...prose(data.lead,width),'',
    ...data.sections.flatMap(([title,lines])=>[...box(title,lines,width),''])];
  if(variant==='B')return [...prose(data.title,width),'',...prose(data.lead,width),'',
    ...data.sections.flatMap(([title,lines])=>[...prose(title,width),...lines.flatMap(line=>prose(line,width)),''])];
  return [...prose(data.title,width),...prose(data.lead,width),'',
    ...data.sections.flatMap(([title,lines])=>[...prose('── '+title+' ──',width),...lines.flatMap(line=>prose(line,width)),''])];
}
function decisionPage(id,variant,width){
  if(id==='discard'){
    const message=['Discard this registration draft?','work-laptop → lab-box','No SSH operation has started.'];
    const actions=['> Keep editing','  Discard draft','Esc also keeps your draft.'];
    if(variant==='A')return [...box('Unsaved draft',message,width),'',...box('Choose',actions,width)];
    if(variant==='B')return [...prose(message[0],width),'',...prose(message[1],width),...prose(message[2],width),'',...actions.flatMap(x=>prose(x,width))];
    return ['Draft / register public key','',...pairs([['Destination','lab-box'],['Key','work-laptop'],['Execution','Not started']],width),'',...actions.flatMap(x=>prose(x,width))];
  }
  if(id==='handoff'){
    const lead=['Continue in the terminal','The TUI will pause for exact checks and approval.'];
    const facts=['Target: lab-box · work-laptop','OpenSSH owns authentication prompts.','Enter passwords/passphrases only there.','Read and approve the exact proposed operation.','The TUI returns with the recorded result.'];
    if(variant==='A')return [...box('Terminal handoff',lead,width),'',...box('What happens next',facts,width),'','> Continue to terminal','  Back to review'];
    if(variant==='B')return [...lead.flatMap(x=>prose(x,width)),'',...facts.flatMap(x=>prose(x,width)),'','> Continue to terminal','  Back to review'];
    return ['Execution / terminal handoff','',...pairs([['Target','lab-box / work-laptop'],['Approval','Required in terminal'],['Secrets','Terminal only'],['Return','Result page']],width),'',...prose('> Continue to terminal · Esc review',width)];
  }
  const facts=[['Destination','lab-box · sam@192.0.2.24:22'],['Public key','work-laptop · ref …7mQ2'],['Authentication','Existing key'],['SSH alias','Do not create']];
  const effect=['Send only this public key to the destination.','Then test SSH authentication with this key.','Exact target checks and approval happen in the terminal.'];
  if(width<50){
    const summary=['Register work-laptop','at lab-box · sam@192.0.2.24:22','No changes yet.'];
    const rest=['',...pairs(facts,width),'',...effect.flatMap(x=>prose(x,width))];
    if(variant==='A')return [...box('Review',summary,width),...prose('> Continue to terminal',width),...rest];
    if(variant==='B')return ['Register this public key?','',...summary,...prose('> Continue to terminal',width),...rest];
    return ['Review / register',...summary,'',...prose('> Continue to terminal',width),...rest];
  }
  if(variant==='A')return [...box('Review registration',['No changes have run yet.'],width),'',
    ...box('Target and options',facts.map(([k,v])=>k+': '+v),width),'',...box('Expected changes',effect,width),'','> Continue to terminal','  Edit draft · Cancel'];
  if(variant==='B')return ['Ready to register this public key?','',...prose('work-laptop → lab-box',width),...prose('sam@192.0.2.24:22',width),'',
    'What will happen',...effect.flatMap(x=>prose(x,width)),'','> Continue to terminal','  Edit draft · Cancel','',...pairs(facts,width)];
  return ['Review / register public key','No changes have run yet.','',...pairs(facts,width),'',
    'Expected effects',...effect.flatMap(x=>prose(x,width)),'',...prose('> Continue to terminal · Edit · Cancel',width)];
}
const edges={
  empty:{title:'No destinations yet',description:'Add a destination to give a login target a readable name.',action:'a Add your first destination',facts:[['Example','lab-box'],['Login','sam@192.0.2.24:22']]},
  'no-tasks':{title:'No saved tasks',description:'KeyFlow has no unfinished work to show. This does not confirm server or registration health.',action:'Tab Browse destinations',facts:[['Saved tasks','0'],['Live SSH check','Not performed']]},
  search:{title:'No matches for "archive"',description:'Your saved destinations have not changed. Clear the search to see all records.',action:'Esc Clear search',facts:[['Search','archive'],['Matches','0 of 3']]},
  stale:{title:'Could not refresh saved records',description:'The state file could not be read. Previously shown records may be out of date. Editing and SSH actions are disabled.',action:'F5 Try refresh again',facts:[['Read error','Permission denied'],['Data freshness','Unknown'],['Actions','Read-only until refresh']]},
  blocked:{title:'The saved key reference is missing',description:'This operation refers to a key that is no longer in saved records. KeyFlow cannot safely choose a replacement.',action:'v Inspect saved references',facts:[['Destination','lab-box'],['Missing key','key-removed-02'],['Continue','Unavailable']]},
  small:{title:'Make the terminal larger',description:'The real TUI needs at least 32 columns and 12 rows. Your draft stays in memory while you resize.',action:'Resize to continue · q Exit',facts:[['Minimum','32 × 12'],['Draft','Retained'],['Operations','Paused']]}
};
function edgePage(id,variant,width){
  const data=edges[id];
  if(width<50)return [...prose(data.title,width),...prose('> '+data.action,width),'',
    ...(variant==='A'?box('Details',[data.description],width):variant==='B'?prose(data.description,width):pairs(data.facts,width)),
    '',...(variant==='C'?prose(data.description,width):pairs(data.facts,width))];
  if(variant==='A')return [...box(data.title,[data.description],width),'',...pairs(data.facts,width),'',...prose('> '+data.action,width)];
  if(variant==='B')return [...prose(data.title,width),'',...prose(data.description,width),'',...prose('> '+data.action,width),'',...pairs(data.facts,width)];
  return [...prose(data.title,width),'',...pairs(data.facts,width),'',...prose(data.description,width),'',...prose('NEXT  '+data.action,width)];
}
function content(page,variant,width){
  switch(page.type){
    case'list':case'picker':return listPage(page.id,variant,width);
    case'detail':return detailPage(page.id,variant,width);
    case'form':return formPage(page.id,variant,width);
    case'result':return resultPage(page.id,variant,width);
    case'document':return documentPage(page.id,variant,width);
    case'edge':return edgePage(page.id,variant,width);
    case'scenario':return scenarioPage(page.id,variant,width);
    default:return decisionPage(page.id,variant,width);
  }
}
function header(page,variant,width){
  if(width<50)return [pad('KeyFlow / '+page.title,width),'─'.repeat(width)];
  if(variant==='A')return [pad('SSH KeyFlow   /   '+page.title,width),'─'.repeat(width)];
  if(variant==='B')return [pad('SSH KeyFlow   ›   '+page.title,width),''];
  return [pad('KEYFLOW  '+page.group.toUpperCase()+' / '+page.title,width),'═'.repeat(width)];
}
function footer(page,width,offset,maxOffset){
  const position=maxOffset ? `${Math.round(offset/maxOffset*100)}%` : 'all';
  let control='↑↓ scroll · Esc back · ? help';
  if(['list','picker'].includes(page.type))control='↑↓ select · Enter open · / find';
  if(page.type==='form')control='Tab field · Enter next · Esc back';
  if(page.id==='review')control='Enter terminal · e edit · Esc back';
  if(page.id==='discard')control='Enter keep editing · d discard';
  if(page.type==='detail')control=page.id==='key'?'r register · v details · h history':'s review work · v details · h history';
  if(page.type==='result')control=page.id==='cancelled'?'Enter destinations · Esc back':
    page.id==='unknown'?'s review work · o output · Enter destination':
    page.id==='manual'?'o instructions · Enter destination':'o output · Enter registration';
  if(page.id==='include')control='o instructions · Enter registration · Esc';
  if(page.id==='config-only')control='v evidence · Esc back · no continuation';
  if(page.id==='stale')control='F5 retry refresh · Esc back · read-only';
  if(width<50){
    if(page.type==='detail')control=page.id==='key'?'r register · v details · Esc':'s review · v details · Esc';
    else if(page.type==='form')control='Tab field · Enter next · Esc';
    else if(page.type==='list'||page.type==='picker')control='↑↓ select · Enter open · Esc';
    else if(page.id==='review')control='Enter terminal · e edit · Esc';
    else if(page.id==='discard')control='Enter keep · d discard · Esc';
    else if(page.type==='result')control=page.id==='cancelled'?'Enter destinations · Esc':
      page.id==='unknown'?'s review · o output · Enter':
      page.id==='manual'?'o instructions · Enter target':'o output · Enter target · Esc';
    else if(page.id==='config-only')control='v evidence · Esc · read-only';
    else if(page.id==='stale')control='F5 retry · Esc · read-only';
    else if(page.id==='include')control='o instructions · Enter target';
    else control='↑↓ scroll · Esc back · ? help';
  }
  return [cut(control,width),cut(`PgUp/PgDn scroll · ${position} · demo`,width)];
}
function frame(page,variant,width,height,offset){
  const body=content(page,variant,width);
  const capacity=height-5;
  const maxOffset=Math.max(0,body.length-capacity);
  offset=Math.max(0,Math.min(offset,maxOffset));
  const visible=body.slice(offset,offset+capacity);
  while(visible.length<capacity)visible.push('');
  return {lines:[...header(page,variant,width),...visible,'─'.repeat(width),...footer(page,width,offset,maxOffset)].map(line=>pad(line,width)),
    offset,maxOffset,body};
}
function lineClass(line){
  const trimmed=line.trim();
  if(/^>/.test(trimmed)||/\[Next\]|\[Continue/.test(trimmed))return'selected';
  if(/unknown|unconfirmed|unfinished|not captured|needs review|Not checked|not confirm|may already/i.test(trimmed))return'warning';
  if(/failed|Error:|missing|could not|Permission denied|Unavailable|disabled/i.test(trimmed))return'danger';
  if(/recorded complete|Passed during/i.test(trimmed))return'good';
  if(/^(┌|└|─|═)/.test(trimmed))return'dim';
  if(/^(SSH KeyFlow|KeyFlow|KEYFLOW|NEXT|What |Other |Already |Ready |No destinations)/.test(trimmed))return'heading';
  return'';
}
function syncUrl(){
  const params=new URLSearchParams({variant:state.variant,page:state.page,size:state.size});
  if(!state.color)params.set('color','off');
  if(state.compare)params.set('compare','all');
  history.replaceState(null,'','?'+params);
}
function render(){
  syncUrl();
  const page=pages.find(p=>p.id===state.page),[width,height]=state.size.split('x').map(Number);
  $('page').value=state.page;$('size').value=state.size;$('color').checked=state.color;$('compare').checked=state.compare;
  document.querySelectorAll('[data-page]').forEach(button=>button.classList.toggle('active',button.dataset.page===state.page));
  document.querySelectorAll('[data-variant]').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.variant===state.variant)));
  $('description').replaceChildren();
  const title=document.createElement('h2'),description=document.createElement('p');
  title.textContent=state.compare?`${page.title} — compare the three structures`:`${state.variant} · ${concepts[state.variant].name}`;
  description.textContent=state.compare?'Same fictional record, same terminal size. Use the page selector to compare another screen.':concepts[state.variant].description;
  $('description').append(title,description);
  $('notice').hidden=!state.notice;$('notice').textContent=state.notice;
  const active=document.activeElement?.dataset?.terminal;
  $('previews').replaceChildren();
  for(const variant of state.compare?['A','B','C']:[state.variant]){
    const drawn=frame(page,variant,width,height,state.offsets[variant]);state.offsets[variant]=drawn.offset;
    const figure=document.createElement('figure');figure.className='preview';
    const caption=document.createElement('figcaption');
    caption.textContent=`${variant} · ${concepts[variant].name}   /   ${page.title}   /   ${width} × ${height}`;
    const wrapElement=document.createElement('div');wrapElement.className='terminal-wrap';
    const terminal=document.createElement('pre');terminal.className='terminal'+(state.color?'':' mono');terminal.tabIndex=0;
    terminal.dataset.terminal=variant;terminal.setAttribute('aria-label',`${variant}: ${page.title}, ${width} columns and ${height} rows`);
    drawn.lines.forEach((line,index)=>{
      const span=document.createElement('span');span.className=lineClass(line);span.textContent=line+(index<drawn.lines.length-1?'\n':'');terminal.append(span);
    });
    terminal.addEventListener('keydown',event=>terminalKey(event,variant));
    terminal.addEventListener('wheel',event=>{event.preventDefault();scroll(variant,event.deltaY>0?2:-2);},{passive:false});
    wrapElement.append(terminal);
    const scrollControls=document.createElement('div');scrollControls.className='scroll-controls';
    for(const [label,amount]of [['↑ Scroll',-3],['↓ Scroll',3]]){
      const button=document.createElement('button');button.textContent=label;button.disabled=amount<0?drawn.offset===0:drawn.offset===drawn.maxOffset;
      button.addEventListener('click',()=>scroll(variant,amount));scrollControls.append(button);
    }
    figure.append(caption,wrapElement,scrollControls);$('previews').append(figure);
  }
  if(active)document.querySelector(`[data-terminal="${active}"]`)?.focus({preventScroll:true});
}
function scroll(variant,amount){state.offsets[variant]+=amount;render();}
function navigate(id){
  if(id===state.page)return;
  state.trail.push(state.page);state.page=id;state.offsets={A:0,B:0,C:0};state.selected=0;state.field=0;state.notice='';render();
}
function choose(variant){
  state.variant=variant;render();
  if(state.compare)document.querySelector(`[data-terminal="${variant}"]`)?.closest('.preview').scrollIntoView({block:'start'});
}
function cycle(amount){choose(['A','B','C'][(['A','B','C'].indexOf(state.variant)+amount+3)%3]);}
function back(){const id=state.trail.pop();if(id){state.page=id;state.offsets={A:0,B:0,C:0};state.field=0;state.selected=0;render();}}
function terminalKey(event,variant){
  const page=pages.find(p=>p.id===state.page),key=event.key;
  if(key==='Tab'&&event.shiftKey){event.preventDefault();$('page').focus();return;}
  if(['ArrowDown','ArrowUp','PageDown','PageUp','Enter','Escape','Tab','Home','End','F5','?','r','s','v','h','o','e','d','a','c','i','u','/','q'].includes(key))event.preventDefault();
  const standalone=()=>{state.notice='Standalone screen preview; use Page to inspect another screen. No operation is performed.';render();};
  if(key==='Escape'||key==='q'){
    if(page.id==='search'){navigate('destinations');return;}
    back();return;
  }
  if(key==='?'){
    if(page.id==='destination')navigate('help');else standalone();
    return;
  }
  // Left/Right remain available to the proposed TUI; the outer switcher changes concepts.
  if(key==='ArrowLeft'||key==='ArrowRight')return;
  if(key==='F5'){
    state.notice='Simulated refresh only. This example keeps its shown state; use Page to inspect another state.';render();return;
  }
  if(key==='PageDown'||key==='PageUp'){scroll(variant,key==='PageDown'?6:-6);return;}
  if(key==='Home'){state.offsets[variant]=0;render();return;}
  if(key==='End'){state.offsets[variant]=999;render();return;}
  if(key==='ArrowDown'||key==='ArrowUp'){
    const direction=key==='ArrowDown'?1:-1;
    if(lists[page.id]){
      state.selected=(state.selected+direction+lists[page.id].rows.length)%lists[page.id].rows.length;
      // Reveal selection in the synthetic preview without changing target identity.
      const [width,height]=state.size.split('x').map(Number),body=content(page,variant,width);
      const selectedLine=body.findIndex(line=>/(?:^|│ )> /.test(line));
      if(selectedLine>=0)state.offsets[variant]=Math.max(0,selectedLine-Math.floor((height-5)/2));
      render();
    }else scroll(variant,direction);
    return;
  }
  if(key==='Tab'){
    if(page.type==='list')navigate(['destinations','keys','registrations','tasks'][(['destinations','keys','registrations','tasks'].indexOf(page.id)+1)%4]);
    else if(page.type==='form'){
      if(page.id==='delete'){standalone();return;}
      state.field++;
      const [width,height]=state.size.split('x').map(Number);
      for(const direction of ['A','B','C']){
        const body=content(page,direction,width),focused=body.findIndex(line=>/(?:^|│ )> /.test(line));
        if(focused>=0&&(focused<state.offsets[direction]||focused>=state.offsets[direction]+height-5))state.offsets[direction]=Math.max(0,focused-2);
      }
      render();
    }
    return;
  }
  if(key==='Enter'){
    if(lists[page.id]&&state.selected!==0){
      state.notice='This row is a visual placeholder. Select the first row for the sample walkthrough, or choose any screen in the page menu.';
      render();return;
    }
    if(page.type==='list')navigate({destinations:'destination',keys:'key',registrations:'registration',tasks:'task'}[page.id]);
    else if(page.type==='picker')navigate(page.id==='recovery-picker'?'resume':page.id==='registration-picker'?'unregister':'register');
    else if(page.type==='form'){
      if(['add','access','register'].includes(page.id))navigate(forms[page.id].next);
      else standalone();
    }
    else if(page.id==='review')navigate('handoff');
    else if(page.id==='handoff')standalone();
    else if(page.id==='discard')back();
    else if(page.type==='result'){
      if(page.id==='cancelled')navigate('destinations');
      else if(['manual','unknown'].includes(page.id))navigate('destination');
      else standalone();
    }
    else standalone();
    return;
  }
  const routes={
    destinations:{a:'add',r:'picker',u:'registration-picker','/':'search'},
    keys:{c:'create',i:'import'},
    registrations:{r:'picker',u:'registration-picker'},
    destination:{s:'resume',r:'picker',u:'unregister',v:'technical',h:'history'},
    key:{r:'destination-picker',d:'delete'},
    registration:{s:'resume',u:'unregister'},
    task:{s:'resume'},
    review:{e:'register'},
    discard:{d:'destinations'},
    unknown:{s:'resume',o:'session-output'},
    manual:{o:'instructions'},
    success:{o:'session-output'},partial:{o:'session-output'},failed:{o:'session-output'},
    include:{o:'include-instructions'}
  };
  // Unsupported example links never open a different target or an execution draft.
  const route=routes[page.id]?.[key];
  if(route)navigate(route);
  else if(['s','v','h','o','e','d','a','c','i','u','r','/'].includes(key))standalone();
}
const groupNames=[...new Set(pages.map(page=>page.group))];
for(const group of groupNames){
  const optgroup=document.createElement('optgroup');optgroup.label=group;
  const label=document.createElement('div');label.className='group-title';label.textContent=group;$('page-links').append(label);
  for(const page of pages.filter(page=>page.group===group)){
    const option=document.createElement('option');option.value=page.id;option.textContent=page.title;optgroup.append(option);
    const button=document.createElement('button');button.className='page-link';button.dataset.page=page.id;button.textContent=page.title;button.addEventListener('click',()=>navigate(page.id));$('page-links').append(button);
  }
  $('page').append(optgroup);
}
$('page').addEventListener('change',event=>navigate(event.target.value));
$('size').addEventListener('change',event=>{state.size=event.target.value;state.offsets={A:0,B:0,C:0};render();});
$('color').addEventListener('change',event=>{state.color=event.target.checked;render();});
$('compare').addEventListener('change',event=>{state.compare=event.target.checked;render();});
$('reset').addEventListener('click',()=>{state.offsets={A:0,B:0,C:0};render();});
$('previous').addEventListener('click',()=>cycle(-1));$('next').addEventListener('click',()=>cycle(1));
document.querySelectorAll('[data-variant]').forEach(button=>button.addEventListener('click',()=>choose(button.dataset.variant)));
document.addEventListener('keydown',event=>{
  if(event.defaultPrevented||event.target.closest('.terminal')||event.target.matches('input,select,textarea,[contenteditable]'))return;
  if(event.key==='ArrowLeft'||event.key==='ArrowRight'){event.preventDefault();cycle(event.key==='ArrowLeft'?-1:1);}
});
// Deliberately exposed for read-only artifact checks in the browser.
window.keyflowPrototype={pages,concepts,frame,content,state};
render();
