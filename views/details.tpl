{{template "header.tpl" .}}
<a href="javascript:history.back()">← Back</a>
<article class="details">
  <img src="{{.Event.ImageURL}}" alt="{{.Event.Name}}">
  <h1>{{.Event.Name}}</h1>
  <p><strong>Date:</strong> {{.Event.Date}}</p>
  <p><strong>Venue:</strong> {{.Event.Venue}}</p>
  {{if .Event.Info}}<p>{{.Event.Info}}</p>{{end}}

  {{if .ShowTickets}}
    <a class="btn" href="/redirect/{{.Event.ID}}">View Tickets</a>
  {{else}}
    <p class="unavailable">Ticket action unavailable.</p>
  {{end}}
</article>
{{template "footer.tpl" .}}