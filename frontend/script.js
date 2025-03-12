const API_BASE_URL = window.location.origin;

const statusDot = document.getElementById("statusDot");
const statusText = document.getElementById("statusText");
const streamOptions = document.getElementById("streamOptions");
const customStreamInput = document.getElementById("customStream");
const setCustomStreamBtn = document.getElementById("setCustomStream");
const volumeSlider = document.getElementById("volumeSlider");
const volumeValue = document.getElementById("volumeValue");
const restartStreamBtn = document.getElementById("restartStream");
const guildIdInput = document.getElementById("guildId");
const voiceChannelIdInput = document.getElementById("voiceChannelId");
const updateVoiceBtn = document.getElementById("updateVoice");
const notification = document.getElementById("notification");
const notificationTitle = document.getElementById("notificationTitle");
const notificationMessage = document.getElementById("notificationMessage");
const notificationIcon = document.getElementById("notificationIcon");
const notificationClose = document.getElementById("notificationClose");
const loadingBar = document.getElementById("loadingBar");
const musicVisualizer = document.getElementById("musicVisualizer");

let botStatus = {
    connected: false,
    streaming: false,
    stream_url: "",
    volume: 0.5,
    guild_id: "",
    voice_channel_id: "",
};

function showNotification(type, title, message) {
    notificationTitle.textContent = title;
    notificationMessage.textContent = message;

    notification.className = "notification " + type;

    if (type === "success") {
        notificationIcon.innerHTML = `
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="#43b581">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                `;
    } else if (type === "error") {
        notificationIcon.innerHTML = `
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="#f04747">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                `;
    } else {
        notificationIcon.innerHTML = `
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="#00b0f4">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                `;
    }

    notification.classList.add("show");

    setTimeout(() => {
        notification.classList.remove("show");
    }, 5000);
}

function updateLoadingBar(visible) {
    if (visible) {
        loadingBar.style.width = "70%";
    } else {
        loadingBar.style.width = "100%";
        setTimeout(() => {
            loadingBar.style.transition = "none";
            loadingBar.style.width = "0";
            setTimeout(() => {
                loadingBar.style.transition = "width 0.3s ease";
            }, 50);
        }, 300);
    }
}

function updateStatusIndicator() {
    if (botStatus.connected) {
        statusDot.className = "status-dot online";
        statusText.textContent = botStatus.streaming
            ? "Streaming"
            : "Connected";

        if (botStatus.streaming) {
            musicVisualizer.style.display = "flex";
        } else {
            musicVisualizer.style.display = "none";
        }
    } else {
        statusDot.className = "status-dot";
        statusText.textContent = "Disconnected";
        musicVisualizer.style.display = "none";
    }
}

function updateStreamOptions() {
    const options = streamOptions.querySelectorAll(".stream-option");
    options.forEach((option) => {
        if (option.dataset.url === botStatus.stream_url) {
            option.classList.add("active");
        } else {
            option.classList.remove("active");
        }
    });
}

async function fetchBotStatus() {
    try {
        updateLoadingBar(true);
        console.log("Fetching bot status from:", `${API_BASE_URL}/status`);
        const response = await fetch(`${API_BASE_URL}/status`);

        if (!response.ok) {
            console.error("Status response not OK:", response.status);
            throw new Error("Failed to fetch status");
        }

        const data = await response.json();
        console.log("Received status data:", data);
        botStatus = data;

        updateStatusIndicator();
        updateStreamOptions();
        guildIdInput.value = data.guild_id;
        voiceChannelIdInput.value = data.voice_channel_id;
        volumeSlider.value = data.volume;
        volumeValue.textContent = `${Math.round(data.volume * 100)}%`;
        updateVolumeSliderBackground();

        return data;
    } catch (error) {
        console.error("Error fetching status:", error);
        showNotification(
            "error",
            "Connection Error",
            "Could not connect to the server",
        );
    } finally {
        updateLoadingBar(false);
    }
}

async function updateVolume(value) {
    try {
        console.log("Updating volume to:", value, "Type:", typeof value);
        updateLoadingBar(true);
        const valueNum = parseFloat(value);

        const response = await fetch(`${API_BASE_URL}/volume`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({ volume: valueNum }),
        });

        console.log("Volume update response status:", response.status);

        if (!response.ok) {
            const errorText = await response.text();
            console.error("Server error response:", errorText);
            throw new Error(
                `Failed to update volume: ${response.status} ${errorText}`,
            );
        }

        showNotification(
            "success",
            "Volume Updated",
            `Volume set to ${Math.round(valueNum * 100)}%`,
        );
        return await response.json();
    } catch (error) {
        console.error("Error updating volume:", error);
        showNotification("error", "Update Failed", "Could not update volume");
    } finally {
        updateLoadingBar(false);
    }
}

async function updateStreamURL(url) {
    try {
        console.log("Updating stream URL to:", url);
        updateLoadingBar(true);
        statusDot.className = "status-dot connecting";
        statusText.textContent = "Connecting...";

        const response = await fetch(`${API_BASE_URL}/stream`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({ url: url }),
        });

        console.log("Stream URL update response status:", response.status);

        if (!response.ok) {
            const errorText = await response.text();
            console.error("Server error response:", errorText);
            throw new Error(
                `Failed to update stream URL: ${response.status} ${errorText}`,
            );
        }

        showNotification(
            "success",
            "Stream Updated",
            "Now playing from new source",
        );

        botStatus.stream_url = url;
        updateStreamOptions();

        setTimeout(fetchBotStatus, 2000);

        return await response.json();
    } catch (error) {
        console.error("Error updating stream URL:", error);
        showNotification(
            "error",
            "Update Failed",
            "Could not update stream source",
        );
        fetchBotStatus();
    } finally {
        updateLoadingBar(false);
    }
}

async function updateVoiceChannel(guildId, voiceId) {
    try {
        console.log("Updating voice channel to:", guildId, voiceId);
        updateLoadingBar(true);
        statusDot.className = "status-dot connecting";
        statusText.textContent = "Connecting...";

        const response = await fetch(`${API_BASE_URL}/voice`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                guild_id: guildId,
                voice_channel_id: voiceId,
            }),
        });

        console.log("Voice channel update response status:", response.status);

        if (!response.ok) {
            const errorText = await response.text();
            console.error("Server error response:", errorText);
            throw new Error(
                `Failed to update voice channel: ${response.status} ${errorText}`,
            );
        }

        showNotification(
            "success",
            "Voice Channel Updated",
            "Bot is connecting to new channel",
        );

        setTimeout(fetchBotStatus, 2000);

        return await response.json();
    } catch (error) {
        console.error("Error updating voice channel:", error);
        showNotification(
            "error",
            "Update Failed",
            "Could not update voice channel",
        );
        fetchBotStatus();
    } finally {
        updateLoadingBar(false);
    }
}

async function restartStream() {
    try {
        console.log("Restarting stream");
        updateLoadingBar(true);
        statusDot.className = "status-dot connecting";
        statusText.textContent = "Reconnecting...";

        const response = await fetch(`${API_BASE_URL}/restart`, {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({}),
        });

        console.log("Restart stream response status:", response.status);

        if (!response.ok) {
            const errorText = await response.text();
            console.error("Server error response:", errorText);
            throw new Error(
                `Failed to restart stream: ${response.status} ${errorText}`,
            );
        }

        showNotification(
            "success",
            "Stream Restarted",
            "Bot is reconnecting to the stream",
        );

        setTimeout(fetchBotStatus, 2000);

        return await response.json();
    } catch (error) {
        console.error("Error restarting stream:", error);
        showNotification(
            "error",
            "Restart Failed",
            "Could not restart the stream",
        );
        fetchBotStatus();
    } finally {
        updateLoadingBar(false);
    }
}

function updateVolumeSliderBackground() {
    const value = volumeSlider.value;
    const percentage = value * 100;
    volumeSlider.style.background = `linear-gradient(to right, var(--primary) 0%, var(--primary) ${percentage}%, rgba(255, 255, 255, 0.1) ${percentage}%, rgba(255, 255, 255, 0.1) 100%)`;
}

volumeSlider.addEventListener("input", function () {
    const value = this.value;
    volumeValue.textContent = `${Math.round(value * 100)}%`;
    updateVolumeSliderBackground();
});

volumeSlider.addEventListener("change", function () {
    const value = parseFloat(this.value);
    console.log("Volume value to send:", value);
    updateVolume(value);
});

streamOptions.addEventListener("click", function (e) {
    const option = e.target.closest(".stream-option");
    if (!option) return;

    const url = option.dataset.url;
    if (url && url !== botStatus.stream_url) {
        updateStreamURL(url);
    }
});

setCustomStreamBtn.addEventListener("click", function () {
    const url = customStreamInput.value.trim();
    if (url) {
        updateStreamURL(url);
    } else {
        showNotification(
            "error",
            "Invalid URL",
            "Please enter a valid stream URL",
        );
    }
});

updateVoiceBtn.addEventListener("click", function () {
    const guildId = guildIdInput.value.trim();
    const voiceId = voiceChannelIdInput.value.trim();

    if (!guildId || !voiceId) {
        showNotification(
            "error",
            "Missing Values",
            "Please enter both Guild ID and Voice Channel ID",
        );
        return;
    }

    updateVoiceChannel(guildId, voiceId);
});

restartStreamBtn.addEventListener("click", function () {
    restartStream();
});

notificationClose.addEventListener("click", function () {
    notification.classList.remove("show");
});

document.addEventListener("DOMContentLoaded", function () {
    console.log("DOM loaded, fetching initial status");
    fetchBotStatus();

    setInterval(fetchBotStatus, 10000);
});
