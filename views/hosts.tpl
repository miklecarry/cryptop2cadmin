<div class="d-flex justify-content-between align-items-center mb-3">
    <h3>Список хостов</h3>
    {{if eq .Role "superadmin"}}
    <a href="/users" class="btn btn-primary">Пользователи</a>
    {{end}}
</div>

<meta http-equiv="refresh" content="15">
{{if .Hosts}}
<div class="row">
    {{range .Hosts}}
    <div class="col-md-4 col-sm-6 mb-3">
        <div class="card shadow-sm 
            {{if not .Online}}border-secondary{{else if .Enabled}}border-success{{else}}border-warning{{end}}">
            <div class="card-body">
                <h5 class="card-title">
                    {{.Host.Name}}
                    {{if not .Host.Active}}
                        <span class="badge bg-secondary">Неaктивен</span>
                    {{else}}
                        <span class="badge bg-success">Aктивен</span>


                    {{end}}
                </h5>
<p class="card-text text-muted small">

    Лимиты: {{.Host.MinLimit}} – {{.Host.MaxLimit}}<br>

</p>
{{if or 
    (eq $.Role "admin") 
    (eq $.Role "superadmin") 
    (and 
        (eq $.Role "user") 
        .Host.User 
        (eq .Host.User.Id $.CurrentUserID)
    )
}}
    <a href="/host/{{.Host.Id}}" class="btn btn-sm btn-outline-primary">Настройки</a>
{{end}}
            </div>
        </div>
    </div>
    {{end}}
</div>
{{else}}
<div class="alert alert-info text-center">
    Пока нет добавленных хостов 🚀
</div>
{{end}}
<div class="d-flex justify-content-between align-items-center mb-3">

    {{if or (eq .Role "admin") (eq .Role "superadmin")}}
        <a href="/host/create" class="btn btn-success">➕ Добавить хост</a>
    {{end}}

</div>