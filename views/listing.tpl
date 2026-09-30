{{template "header.tpl" .}}
{{if .Error}}
  <div class="error">{{.Error}}</div>
{{else}}
  <h1>Events in {{.City}}, {{.CountryCode}}</h1>
  <p class="cache-note">Music cache: {{.Music.Cache}} · Sports cache: {{.Sports.Cache}}</p>

  <section class="section">
    <h2>🎵 Music</h2>
    {{if .Music.Err}}<div class="error">{{.Music.Err}}</div>
    {{else if not .Music.Events}}<div class="empty">No Music events found.</div>
    {{else}}<div class="grid">
      {{range .Music.Events}}
        <article class="card">
          <img src="{{.ImageURL}}" alt="{{.Name}}">
          <h3>{{.Name}}</h3>
          <p>{{.Date}}</p>
          <p>{{.Venue}}</p>
          <a href="/events/{{.ID}}">View Details</a>
        </article>
      {{end}}
    </div>{{end}}
  </section>

  <section class="section">
    <h2>🏅 Sports</h2>
    {{if .Sports.Err}}<div class="error">{{.Sports.Err}}</div>
    {{else if not .Sports.Events}}<div class="empty">No Sports events found.</div>
    {{else}}<div class="grid">
      {{range .Sports.Events}}
        <article class="card">
          <img src="{{.ImageURL}}" alt="{{.Name}}">
          <h3>{{.Name}}</h3>
          <p>{{.Date}}</p>
          <p>{{.Venue}}</p>
          <a href="/events/{{.ID}}">View Details</a>
        </article>
      {{end}}
    </div>{{end}}
  </section>
{{end}}
{{template "footer.tpl" .}}