<style>
.user-card {
    transition: transform 0.2s;
}
.user-card:hover {
    transform: translateY(-2px);
}
</style>

<div class="d-flex justify-content-between align-items-center mb-4">
    <h3>Управление пользователями</h3>
    <a href="/" class="btn btn-outline-secondary">&larr; Назад</a>
</div>

<!-- Сообщения -->
{{if .Error}}
<div class="alert alert-danger alert-dismissible fade show" role="alert">
    {{.Error}}
    <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
</div>
{{end}}

<!-- Форма добавления -->
<div class="card mb-4">
    <div class="card-header bg-primary text-white">
        <h5 class="mb-0">Добавить нового пользователя</h5>
    </div>
    <div class="card-body">
        <form method="post" action="/users">
            <div class="row">
                <div class="col-md-3 mb-3">
                    <label class="form-label">ФИО</label>
                    <input type="text" name="fullname" class="form-control" required>
                </div>
                <div class="col-md-3 mb-3">
                    <label class="form-label">Логин</label>
                    <input type="text" name="username" class="form-control" required>
                </div>
                <div class="col-md-3 mb-3">
                    <label class="form-label">Пароль</label>
                    <input type="password" name="password" class="form-control" required>
                </div>
                <div class="col-md-2 mb-3">
                    <label class="form-label">Роль</label>
                    <select name="role" class="form-select" required>
                        <option value="admin">Админ</option>
                        <option value="user">Пользователь</option>
                        <!-- superadmin можно создавать только через БД -->
                    </select>
                </div>
                <div class="col-md-1 d-flex align-items-end mb-3">
                    <button type="submit" class="btn btn-success">Добавить</button>
                </div>
            </div>
        </form>
    </div>
</div>

<!-- Список пользователей -->
{{if .Users}}
<div class="row">
    {{range .Users}}
    <div class="col-lg-6 mb-3">
        <div class="card user-card shadow-sm h-100">
            <div class="card-body d-flex justify-content-between align-items-start">
                <div>
                    <h5 class="card-title mb-1">{{.FullName}}</h5>
                    <p class="text-muted mb-1">
                        <strong>{{.Username}}</strong> • {{.Role}}
                    </p>
                </div>
                <div>
                    {{if ne .Role "superadmin"}}
                    <a href="/user/{{.Id}}/delete" 
                       class="btn btn-sm btn-outline-danger"
                       onclick="return confirm('Удалить пользователя {{.FullName}}?')">
                        Удалить
                    </a>
                    {{else}}
                    <span class="badge bg-danger">SuperAdmin</span>
                    {{end}}
                </div>
            </div>
        </div>
    </div>
    {{end}}
</div>
{{else}}
<div class="alert alert-info text-center">
    Пока нет пользователей
</div>
{{end}}