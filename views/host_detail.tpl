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
    <h5 class="mt-4">Управление API-токенами для IP-адресов</h5>
    <p class="text-muted small">Если для IP-адреса задан токен, он будет передаваться хосту вместо общего 'Access Token'. Если токен пустой, используется общий 'Access Token'.</p>

    <!-- Таблица существующих IP-токенов -->
    <div class="mb-3">
        <label class="form-label">Назначенные токены</label>
        <div id="tokensListContainer">
            {{if .HostTokens}}
                {{range $ip, $token := .HostTokens}}
                    <div class="d-flex mb-2 align-items-center" data-ip="{{ $ip }}">
                        <input type="text" class="form-control me-2 ip-input" value="{{ $ip }}" readonly style="flex: 1;">
                        <input type="text" class="form-control me-2 token-input" value="{{ $token }}" placeholder="API Token" style="flex: 2;">
                        <button type="button" class="btn btn-outline-danger btn-sm delete-token-btn" data-ip="{{ $ip }}">Удалить</button>
                    </div>
                {{end}}
            {{else}}
                <p class="text-muted">Токены для IP-адресов не назначены.</p>
            {{end}}
        </div>
    </div>

    <!-- Форма для добавления нового IP-токена -->
    <div class="mb-3">
        <label class="form-label">Добавить/обновить токен для IP</label>
        <div class="d-flex">
            <input type="text" id="newIPInput" class="form-control me-2" placeholder="IP-адрес (например, 192.168.1.100)">
            <input type="text" id="newTokenInput" class="form-control me-2" placeholder="API Token">
            <button type="button" id="addTokenBtn" class="btn btn-outline-primary">Добавить/Обновить</button>
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
    async function saveTokens() {
        const tokensContainer = document.getElementById('tokensListContainer');
        const tokenInputs = tokensContainer.querySelectorAll('.token-input');
        const newTokensMap = {};

        tokenInputs.forEach(input => {
            const ipDiv = input.closest('[data-ip]');
            if (ipDiv) {
                const ip = ipDiv.getAttribute('data-ip');
                const token = input.value.trim(); // Убираем лишние пробелы
                newTokensMap[ip] = token;
            }
        });

        try {
            // Отправляем JSON-объект с новым мапом токенов
            const res = await fetch(`/api/host/${encodeURIComponent(hostId)}/update-tokens`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(newTokensMap),
            });

            if (!res.ok) {
                const errorData = await res.json().catch(() => ({ error: 'Неизвестная ошибка' }));
                throw new Error(errorData.error || `HTTP error! status: ${res.status}`);
            }

            const data = await res.json();
            if (data.status === 'ok') {
                // console.log('Токены успешно обновлены');
                // Можно показать сообщение об успехе
            } else {
                throw new Error(data.error || 'Неизвестная ошибка при сохранении');
            }
        } catch (err) {
            console.error('Ошибка при сохранении токенов:', err);
            alert('Ошибка при сохранении токенов: ' + err.message);
        }
    }

    // Обработчик кнопки "Добавить/Обновить"
    document.getElementById('addTokenBtn').addEventListener('click', () => {
        const ipInput = document.getElementById('newIPInput');
        const tokenInput = document.getElementById('newTokenInput');
        const ip = ipInput.value.trim();
        const token = tokenInput.value.trim();

        if (!ip) {
            alert('Пожалуйста, введите IP-адрес.');
            return;
        }

        const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
        if (!ipRegex.test(ip)) {
            alert('Неверный формат IP-адреса.');
            return;
        }

        const tokensContainer = document.getElementById('tokensListContainer');
        const existingDiv = tokensContainer.querySelector(`[data-ip="${ip}"]`);

        if (existingDiv) {
            existingDiv.querySelector('.token-input').value = token;
        } else {
            const newDiv = document.createElement('div');
            newDiv.className = 'd-flex mb-2 align-items-center';
            // --- КРИТИЧЕСКОЕ ИЗМЕНЕНИЕ: Атрибут data-ip добавляется КОНТЕЙНЕРНОМУ DIV ---
            newDiv.setAttribute('data-ip', ip);
            // Кнопка НЕ должна иметь data-ip
            newDiv.innerHTML = `
                <input type="text" class="form-control me-2 ip-input" value="${escapeHtml(ip)}" readonly style="flex: 1;">
                <input type="text" class="form-control me-2 token-input" value="${escapeHtml(token)}" placeholder="API Token" style="flex: 2;">
                <button type="button" class="btn btn-outline-danger btn-sm delete-token-btn">Удалить</button>
            `;
            tokensContainer.appendChild(newDiv);
        }

        ipInput.value = '';
        tokenInput.value = '';

        saveTokens();
    });

    // Делегирование события клика для кнопок "Удалить"
    document.getElementById('tokensListContainer').addEventListener('click', (e) => {
        if (e.target.classList.contains('delete-token-btn')) {
            const divToRemove = e.target.closest('[data-ip]');

            if (divToRemove) {
                const ipToRemove = divToRemove.getAttribute('data-ip');

                if (confirm(`Вы уверены, что хотите удалить токен для IP ${ipToRemove}?`)) {
                    // Формируем новую мапу, исключая удаляемый IP
                    const tokensContainer = document.getElementById('tokensListContainer');
                    const tokenDivs = tokensContainer.querySelectorAll('[data-ip]');
                    const newTokensMap = {};

                    tokenDivs.forEach(div => {
                        const ip = div.getAttribute('data-ip');
                        // Пропускаем div с IP, который нужно удалить
                        if (ip !== ipToRemove) {
                            const tokenInput = div.querySelector('.token-input');
                            if (tokenInput) {
                                const token = tokenInput.value.trim();
                                newTokensMap[ip] = token;
                            }
                        }
                    });

                    // Отправляем обновлённый JSON без удаляемого IP
                    fetch(`/api/host/${encodeURIComponent(hostId)}/update-tokens`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify(newTokensMap),
                    })
                    .then(res => {
                        if (!res.ok) {
                            return res.json().then(errorData => {
                                const errorMessage = errorData.error || `HTTP error! status: ${res.status}`;
                                throw new Error(errorMessage);
                            });
                        }
                        return res.json();
                    })
                    .then(data => {
                        if (data.status === 'ok') {
                            window.location.reload();
                        } else {
                            throw new Error(data.error || 'Неизвестная ошибка при сохранении');
                        }
                    })
                    .catch(err => {
                        console.error('Ошибка при удалении токена:', err);
                        alert('Ошибка при удалении токена: ' + err.message);
                    });
                }
            }
        }
    });

    // Обработчик изменения токена вручную (для уже существующих записей)
    document.getElementById('tokensListContainer').addEventListener('change', (e) => {
        if (e.target.classList.contains('token-input')) {
            // Сохраняем изменения при изменении любого токена
            // Для производительности можно добавить debounce
            saveTokens();
        }
    });
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