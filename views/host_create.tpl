

<div class="container mt-4">
    <div class="d-flex justify-content-between align-items-center mb-4">
        <h3>Добавить новый хост</h3>
        <a href="/" class="btn btn-outline-secondary">&larr; Назад к списку</a>
    </div>

    {{if .Error}}
        <div class="alert alert-danger">{{.Error}}</div>
    {{end}}

    <form method="post" action="/host/create">
        <div class="mb-3">
            <label class="form-label">Имя хоста (уникальное)</label>
            <input type="text" name="name" class="form-control" required maxlength="100">
            <div class="form-text">Используется для идентификации клиента. Должно быть уникальным.</div>
        </div>

<div class="mb-3">
    <label class="form-label">Адрес сервера (host:port)</label>
    <input type="text" name="server_addr" class="form-control" value="{{.Host.ServerAddr}}" placeholder="192.157.1.34:443">
    <div class="form-text">Используется клиентом для подключения. Формат: host:port</div>
</div>
        <h5 class="mt-4">Конфигурация клиента</h5>
        <div class="mb-3">
    <label class="form-label">Пользователь *</label>
    <select name="user_id" class="form-select" required>
        <option value="">— Выберите пользователя —</option>
        {{range $.Users}}
            <option value="{{.Id}}">{{.Username}} ({{.FullName}})</option>
        {{end}}
    </select>
</div>
        <div class="form-check mb-3">
            <input type="checkbox" name="active" id="active" checked class="form-check-input">
            <label class="form-check-label" for="active">Активен</label>
        </div>

        <div class="mb-3">
            <label class="form-label">Socket URL</label>
            <input type="text" name="socket_url" class="form-control">
        </div>

        <div class="mb-3">
            <label class="form-label">Access Token</label>
            <input type="text" name="access_token" class="form-control">
        </div>


        <div class="row mb-3">
            <div class="col">
                <label class="form-label">Min Amount</label>
                <input type="number" name="min_limit" class="form-control" value="0" min="0">
            </div>
            <div class="col">
                <label class="form-label">Max Amount</label>
                <input type="number" name="max_limit" class="form-control" value="0" min="0">
            </div>
        </div>

        <h5 class="mt-4">Настройки задержек</h5>

        <div class="form-check mb-3">
            <input type="checkbox" name="priority" id="priority" class="form-check-input">
            <label class="form-check-label" for="priority">Приоритетный хост</label>
        </div>

        <div class="mb-3">
            <label class="form-label">Таймаут (секунды)</label>
            <input type="number" name="timeout" class="form-control" value="0" min="0">
            <div class="form-text">StopTime = now + это значение (если >0)</div>
        </div>

        <button type="submit" class="btn btn-success">Создать хост</button>
        <a href="/" class="btn btn-secondary">Отмена</a>
    </form>
</div>

