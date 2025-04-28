const chatID = "ваш_chat_id"; // Замените на реальный chat_id
const userID = "ваш_user_id"; // Замените на реальный user_id

document.getElementById('message-form').addEventListener('submit', function(e) {
    e.preventDefault();
    const messageInput = document.getElementById('message-input');
    sendMessage(messageInput.value);
    messageInput.value = ''; // Очистка поля ввода
});

function sendMessage(content) {
    fetch(`/api/messages/${chatID}/${userID}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: content }) // Не передаем chat_id и user_id в теле запроса
    })
    .then(response => response.json())
    .then(data => {
        addMessageToChat(data);
    });
}

function fetchMessages() {
    fetch(`/api/messages/${chatID}`)
    .then(response => response.json())
    .then(messages => {
        messages.forEach(message => addMessageToChat(message));
    });
}

// Загрузка сообщений при загрузке страницы
fetchMessages();
