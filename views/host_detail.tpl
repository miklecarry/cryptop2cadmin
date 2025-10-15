<div class="d-flex justify-content-between align-items-center mb-4">
    <h3>Настройки хоста: <code>{{.Host.Name}}</code></h3>
    <a href="/" class="btn btn-outline-secondary">&larr; Назад к списку</a>
</div>

<div class="row">
    <div class="col-md-6">
        <h5>Информация о хосте</h5>
        <table class="table table-sm table-bordered">
            <tr>
                <th>Имя (уникальное)</th>
                <td>{{.Host.Name}}</td>
            </tr>
            <tr>
                <th>Текущий IP</th>
                <td>{{.Host.Ip}}</td>
            </tr>
            <tr>
                <th>Лимиты</th>
                <td>{{.Host.MinLimit}} – {{.Host.MaxLimit}}</td>
            </tr>
            <tr>
                <th>Создан</th>
                <td>{{.Host.CreatedAt.Format "2006-01-02 15:04:05"}}</td>
            </tr>
        </table>

        <h5 class="mt-4">Настройки Задержек</h5>
        <form method="post" action="/host/{{.Host.Id}}/update">
            <div class="form-check mb-3">
                <input type="checkbox" name="priority" id="priority" {{if .Host.Priority}}checked{{end}} class="form-check-input">
                <label class="form-check-label" for="priority">
                    Приоритетный хост
                </label>
                <div class="form-text">
                    Не получать Таймаут.
                </div>
            </div>
            <div class="mb-3">
                <label for="timeout" class="form-label">Таймаут (секунды)</label>
                <input type="number" name="timeout" id="timeout" value="{{.Host.Timeout}}" min="0" class="form-control" placeholder="0 — отключено">
                <div class="form-text">
                    Используется только если хост не приоритетный и запущены приоритетные хосты.
                </div>
            </div>
            <button type="submit" class="btn btn-primary">Сохранить настройки</button>
        </form>
    </div>

    <div class="col-md-6">
        <h5>Последние логи ({{len .Logs}})</h5>
        {{if .Logs}}
            <div style="max-height: 600px; overflow-y: auto; border: 1px solid #dee2e6; border-radius: 4px; padding: 10px;">
                {{range .Logs}}
                <div class="alert 
                    {{if eq .Level "err"}}alert-danger{{else if eq .Level "bounty"}}alert-warning{{else}}alert-info{{end}} 
                    p-2 mb-2" style="font-size: 0.85rem;">
                    <strong>[{{.Level}}]</strong> {{.Message}}
                    <br><small class="text-muted">{{.CreatedAt.Format "2006-01-02 15:04:05"}}</small>
                </div>
                {{end}}
            </div>
        {{else}}
            <p class="text-muted">Логи ещё не поступали.</p>
        {{end}}
    </div>
</div>

<!-- Удаление -->
<div class="mt-5 p-3 bg-light border rounded">
    <h5 class="text-danger">Опасная операция</h5>
    <p>Удаление хоста приведёт к:</p>
    <ul>
        <li>Полной потере всех логов этого хоста</li>
        <li>Удалению из списка хостов</li>
        <li>Очистке состояния в памяти</li>
    </ul>
    <a href="/host/{{.Host.Id}}/delete" 
       class="btn btn-danger"
       onclick="return confirm('Вы уверены, что хотите удалить хост \'{{.Host.Name}}\' и все его логи? Это действие нельзя отменить.')">
        Удалить хост
    </a>
</div>