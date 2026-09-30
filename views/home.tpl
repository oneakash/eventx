{{template "header.tpl" .}}
<h1>Find events in your city</h1>
<div class="search-box">
  <input id="cityInput" type="text" placeholder="Start typing a city…" autocomplete="off">
  <button id="searchBtn" disabled>Search</button>
  <ul id="suggestions" class="suggestions"></ul>
</div>
<script src="/static/js/autocomplete.js"></script>
{{template "footer.tpl" .}}