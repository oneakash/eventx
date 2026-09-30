{{template "header.tpl" .}}
<div class="error-page">
  <h1>Something went wrong</h1>
  <p>{{.Error}}</p>
  <a href="/">Return home</a>
</div>
{{template "footer.tpl" .}}