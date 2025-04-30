document.getElementById("message-form").addEventListener("submit", function(event) {
    event.preventDefault();
    
    const messageInput = document.getElementById("message-input");
    const messageContent = messageInput.value;
    
    // Здесь вы можете отправить сообщение на сервер
    // Например, с помощью fetch или XMLHttpRequest

    // Для демонстрации добавим сообщение в чат
    const message = {
        ID: Date.now(), // Используем временную метку как ID
        ChatID: 1, // Пример ID чата
        UserID: "user123", // Пример ID пользователя
        Content: messageContent,
        CreatedAt: new Date()
    };

    renderMessage(message);
    messageInput.value = ""; // Очистить поле ввода
});

function renderMessage(message) {
    const messagesDiv = document.getElementById("messages");
    const messageDiv = document.createElement("div");
    messageDiv.classList.add("message");
    messageDiv.innerHTML = `<span class="user">${message.UserID}</span>: ${message.Content} <small>${new Date(message.CreatedAt).toLocaleTimeString()}</small>`;
    messagesDiv.appendChild(messageDiv);
}
