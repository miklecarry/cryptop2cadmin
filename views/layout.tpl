<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <title>{{.AppName}}</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- Bootstrap 5 -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
</head>
<body class="bg-light">
<nav class="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
    <div class="container-fluid">
        <a class="navbar-brand" href="/">CryptoBot Admin</a>

        <div class="d-flex align-items-center">
            {{if and (ne .CurrentPath "/") (ne .CurrentPath "/login")}}
            <a href="/" class="btn btn-outline-light btn-sm me-2">&larr; Назад</a>
            {{end}}
            {{if ne .CurrentPath "/login"}}
            <a href="/logout" class="btn btn-outline-light btn-sm">Выйти</a>
            {{end}}
        </div>
    </div>
</nav>

    <div class="container">

            {{.LayoutContent}}

    </div>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>