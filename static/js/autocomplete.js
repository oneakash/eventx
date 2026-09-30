(function () {
  const input=document.getElementById('cityInput');
  const list=document.getElementById('suggestions');
  const btn =document.getElementById('searchBtn');
  let sessionToken =crypto.randomUUID();
  let selected =null;
  let debounce;

  input.addEventListener('input', ()=>{
    selected =null;
    btn.disabled =true;
    clearTimeout(debounce);
    debounce=setTimeout(fetchSuggestions,250);
  });

  async function fetchSuggestions(){
    const q=input.value.trim();
    if (q.length < 3) { list.innerHTML = '';return;}
    try {
      const res =await fetch(`/api/locations/autocomplete?input=${encodeURIComponent(q)}&sessionToken=${sessionToken}`);
      const data= await res.json();
      list.innerHTML='';
      (data.suggestions|| []).forEach(s =>{
        const li=document.createElement('li');
        li.textContent =s.text;
        li.addEventListener('click',() => selectSuggestion(s));
        list.appendChild(li);
      });
    } catch (e){console.error(e);}
  }

  async function selectSuggestion(s) {
    input.value=s.text;
    list.innerHTML ='';
    try {
      const res=await fetch(`/api/locations/${encodeURIComponent(s.placeId)}?sessionToken=${sessionToken}`);
      const place=await res.json();
      if (place.city && place.countryCode) {
        selected=place;
        btn.disabled=false;
        sessionToken=crypto.randomUUID(); // new token after selection
      }
    } catch (e){console.error(e);}
  }

  btn.addEventListener('click',()=>{
    if (!selected)return;
    window.location.href=`/events?city=${encodeURIComponent(selected.city)}&countryCode=${encodeURIComponent(selected.countryCode)}`;
  });
})();