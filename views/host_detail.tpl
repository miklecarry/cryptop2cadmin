<div class="d-flex justify-content-between align-items-center mb-4">
    <h3>Настройки хоста: <code>{{.Host.Name}}</code></h3>
    <a href="/" class="btn btn-outline-secondary">&larr; Назад к списку</a>
</div>

<div class="row">
    <div class="col-md-6">
  <h5 class="mt-4">Конфигурация клиента</h5>
<form method="post" action="/host/{{.Host.Id}}/update">
<div class="form-check form-switch">
    <input class="form-check-input" type="checkbox" id="activeSwitch"
           {{if .Host.Active}}checked{{end}}>
    <label class="form-check-label" for="activeSwitch">
        {{if .Host.Active}}Активен{{else}}Неактивен{{end}}
    </label>
</div>
{{if or (eq .Role "admin") (eq .Role "superadmin")}}
    <!-- Привязка к пользователю -->
    <div class="mb-3">
        <label class="form-label">Пользователь *</label>
        <select name="user_id" class="form-select" required>
            {{range $.Users}}
                <option value="{{.Id}}" {{if $.Host.User.Id | eq .Id}}selected{{end}}>
                    {{.Username}} {{if .FullName}}({{.FullName}}){{end}}
                </option>
            {{end}}
        </select>
    </div>

    <!-- Socket URL -->
    <div class="mb-3">
        <label class="form-label">Socket URL</label>
        <input type="text" name="socket_url" class="form-control" value="{{.Host.SocketURL}}">
    </div>

    <!-- Адрес сервера (ServerAddr) -->
    <div class="mb-3">
        <label class="form-label">Адрес сервера (host:port)</label>
        <input type="text" name="server_addr" class="form-control" value="{{.Host.ServerAddr}}" placeholder="192.157.1.34:443">
    </div>
{{end}}

<!-- Min/Max и Logger — доступны всем -->
<div class="row mb-3">
    <div class="col">
        <label class="form-label">Min Amount</label>
        <input type="number" name="min_limit" class="form-control" value="{{.Host.MinLimit}}" min="0">
    </div>
    <div class="col">
        <label class="form-label">Max Amount</label>
        <input type="number" name="max_limit" class="form-control" value="{{.Host.MaxLimit}}" min="0">
    </div>
</div>
<div class="mb-3">
    <label class="form-label">Access Token (для app.cr.bot)</label>
    <input type="text" name="access_token" class="form-control" value="{{.Host.AccessToken}}" placeholder="Введите токен от app.cr.bot">
    <div class="form-text">
        Используется для авторизации запросов к app.cr.bot
    </div>
</div>

{{if or (eq .Role "admin") (eq .Role "superadmin")}}
    <!-- Настройки задержек (только для админов) -->
    <h5 class="mt-4">Настройки задержек</h5>
    <div class="form-check mb-3">
        <input type="checkbox" name="priority" id="priority" {{if .Host.Priority}}checked{{end}} class="form-check-input">
        <label class="form-check-label" for="priority">Приоритетный хост</label>
        <div class="form-text">Не получать Таймаут.</div>
    </div>
    <div class="mb-3">
        <label for="timeout" class="form-label">Таймаут (секунды)</label>
        <input type="number" name="timeout" id="timeout" value="{{.Host.Timeout}}" min="0" class="form-control" placeholder="0 — отключено">
        <div class="form-text">StopTime = now + это значение (если >0)</div>
    </div>
{{end}}

    <button type="submit" class="btn btn-primary">Сохранить настройки</button>
</form>

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

<div id="paymentMethodModal" class="modal" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content">
            <div class="modal-header">
                <h5 class="modal-title">Выберите метод оплаты</h5>
                <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
                <div id="methodsList" class="list-group">
                    <!-- Сюда будет подставлен список -->
                </div>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Отмена</button>
                <button type="button" id="confirmMethodBtn" class="btn btn-primary" disabled>Выбрать</button>
            </div>
        </div>
    </div>
</div>

<script>
document.addEventListener("DOMContentLoaded", () => {
    const activeSwitch = document.getElementById('activeSwitch');
    // Получаем host id из шаблона (вставьте в шаблон data-host-id="{{.Host.Id}}")
    // Если у вас уже есть способ доступиться к Id на странице — используйте его.
    const hostId = '{{.Host.Id}}';

    // Если вы не хотите в шаблоне вставлять hostId как JS-переменную,
    // можно прочитать его из ссылки "Удалить" или других data-атрибутов.
    // Убедитесь, что template заменяет {{.Host.Id}} корректно.

    // Перехват изменения состояния переключателя
    activeSwitch.addEventListener('change', async (e) => {
        const turningOn = activeSwitch.checked;

        // Блокируем переключатель пока идёт проверка
        activeSwitch.disabled = true;

        if (turningOn) {
            try {
                const res = await fetch(`/api/host/${encodeURIComponent(hostId)}/payment-methods`);
                if (!res.ok) throw new Error('Не удалось запросить методы оплаты');
                const data = await res.json();

                // нормализуем список аккаунтов
                const accounts = data.accounts || data.data || (data.result && data.result.accounts) || [];
                if (!accounts || accounts.length === 0) {
                    alert('Нет доступных методов оплаты или токен неверный.');
                    activeSwitch.checked = false;
                    activeSwitch.disabled = false;
                    return;
                }

                // Покажем модалку с готовыми методами и коллбеком
                showPaymentMethodModal(data, hostId, false, async (started) => {
                    if (!started) {
                        // пользователь отменил или был error
                        activeSwitch.checked = false;
                    } else {
                        // успешно запущено — оставить как есть
                        activeSwitch.checked = true;
                    }
                    activeSwitch.disabled = false;
                });

            } catch (err) {
                console.error(err);
                alert('Ошибка при запросе методов оплаты. Хост не активирован.');
                activeSwitch.checked = false;
                activeSwitch.disabled = false;
            }
        } else {
            // Выключаем — отправляем стоп
            try {
                const res = await fetch(`/api/host/${encodeURIComponent(hostId)}/stop-monitoring`, {
                    method: 'POST',
                    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                });
                if (!res.ok) throw new Error('Не удалось остановить мониторинг');
                const json = await res.json();
                if (json.status === 'ok') {
                    activeSwitch.checked = false;
                } else {
                    alert('Ошибка при остановке: ' + (json.error || JSON.stringify(json)));
                    // откатить в true, если остановить не удалось
                    activeSwitch.checked = true;
                }
            } catch (err) {
                console.error(err);
                alert('Не удалось остановить мониторинг на сервере.');
                activeSwitch.checked = true;
            } finally {
                activeSwitch.disabled = false;
            }
        }
    });

    // Обновлённая версия showPaymentMethodModal: даёт возможность выбрать элемент и нажать "Выбрать"
    window.showPaymentMethodModal = function(methods, hostId, currentState, updateUI) {
        const modalEl = document.getElementById('paymentMethodModal');
        const modal = new bootstrap.Modal(modalEl);
        const list = document.getElementById('methodsList');
        const confirmBtn = document.getElementById('confirmMethodBtn');

        list.innerHTML = '';
        confirmBtn.disabled = true;
        confirmBtn.dataset.selected = '';

        let accounts = methods.accounts || methods.data || (methods.result && methods.result.accounts) || [];

        if (!accounts || accounts.length === 0) {
            list.innerHTML = '<div class="list-group-item text-danger">Нет доступных методов оплаты</div>';
            confirmBtn.disabled = true;
            modal.show();
            return;
        }

        accounts.forEach(account => {
            const id = account.id || account.uuid || account.payment_id || '';
            const title = account.brand_name || account.name || id || 'Без названия';
            const sub = account.mcc || account.category || '';

            const item = document.createElement('button');
            item.type = 'button';
            item.className = 'list-group-item list-group-item-action d-flex justify-content-between align-items-center';
            item.innerHTML = `
                <div>
                    <strong>${escapeHtml(title)}</strong><br>
                    <small class="text-muted">${escapeHtml(sub)}</small>
                </div>
                <span class="badge bg-light text-dark">${escapeHtml(id)}</span>
            `;
            item.dataset.methodId = id;
            list.appendChild(item);

            item.addEventListener('click', () => {
                // снять класс у всех
                Array.from(list.children).forEach(ch => ch.classList.remove('active'));
                item.classList.add('active');
                confirmBtn.disabled = false;
                confirmBtn.dataset.selected = id;
            });
        });

        confirmBtn.onclick = async () => {
            const selectedId = confirmBtn.dataset.selected;
            if (!selectedId) return;

            confirmBtn.disabled = true;
            confirmBtn.textContent = 'Запуск...';

            try {
                // POST на запуск мониторинга (контроллер StartMonitoring у вас ожидает method_id и id как param)
                const res = await fetch(`/api/host/${encodeURIComponent(hostId)}/start-monitoring`, {
                    method: 'POST',
                    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
                    body: `method_id=${encodeURIComponent(selectedId)}`
                });
                const json = await res.json();
                if (res.ok && json.status === 'ok') {
                    modal.hide();
                    updateUI(true);
                    // можно показать сообщение
                    // alert('Мониторинг запущен');
                } else {
                    const errMsg = json.error || 'Ошибка запуска';
                    alert('❌ ' + errMsg);
                    updateUI(false);
                }
            } catch (err) {
                console.error(err);
                alert('Не удалось запустить мониторинг (network).');
                updateUI(false);
            } finally {
                confirmBtn.disabled = false;
                confirmBtn.textContent = 'Выбрать';
            }
        };

        modal.show();
    };

    // Небольшая helper-функция для безопасности HTML вставки
    function escapeHtml(text) {
        if (!text && text !== 0) return '';
        return String(text)
          .replaceAll('&', '&amp;')
          .replaceAll('<', '&lt;')
          .replaceAll('>', '&gt;')
          .replaceAll('"', '&quot;')
          .replaceAll("'", '&#039;');
    }
});
</script>