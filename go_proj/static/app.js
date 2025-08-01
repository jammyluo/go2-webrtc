// Go2 机器人控制应用
class Go2RobotController {
    constructor() {
        this.websocket = null;
        this.isConnected = false;
        this.robotIP = '';
        this.token = '';
        
        this.initElements();
        this.initEventListeners();
        this.loadSavedValues();
    }

    initElements() {
        this.elements = {
            robotIP: document.getElementById('robot-ip'),
            token: document.getElementById('token'),
            connectBtn: document.getElementById('connect-btn'),
            disconnectBtn: document.getElementById('disconnect-btn'),
            connectText: document.getElementById('connect-text'),
            statusText: document.getElementById('status-text'),
            connectionStatus: document.getElementById('connection-status'),
            commandSelect: document.getElementById('command-select'),
            sendCommandBtn: document.getElementById('send-command-btn'),
            openVideoBtn: document.getElementById('open-video-btn'),
            closeVideoBtn: document.getElementById('close-video-btn'),
            videoStream: document.getElementById('video-stream'),
            videoPlaceholder: document.getElementById('video-placeholder'),
            logContent: document.getElementById('log-content'),
            notification: document.getElementById('notification'),
            quickCommands: document.querySelectorAll('.quick-command')
        };
    }

    initEventListeners() {
        // 连接按钮
        this.elements.connectBtn.addEventListener('click', () => this.connect());
        this.elements.disconnectBtn.addEventListener('click', () => this.disconnect());

        // 命令按钮
        this.elements.sendCommandBtn.addEventListener('click', () => this.sendCommand());
        this.elements.openVideoBtn.addEventListener('click', () => this.openVideo());
        this.elements.closeVideoBtn.addEventListener('click', () => this.closeVideo());

        // 快速命令按钮
        this.elements.quickCommands.forEach(btn => {
            btn.addEventListener('click', (e) => {
                const command = e.target.dataset.command;
                this.sendQuickCommand(command);
            });
        });

        // 输入框变化时保存
        this.elements.robotIP.addEventListener('change', () => this.saveValues());
        this.elements.token.addEventListener('change', () => this.saveValues());
    }

    loadSavedValues() {
        const savedRobotIP = localStorage.getItem('robotIP');
        const savedToken = localStorage.getItem('token');

        if (savedRobotIP) {
            this.elements.robotIP.value = savedRobotIP;
        }
        if (savedToken) {
            this.elements.token.value = savedToken;
        }
    }

    saveValues() {
        localStorage.setItem('robotIP', this.elements.robotIP.value);
        localStorage.setItem('token', this.elements.token.value);
    }

    connect() {
        this.robotIP = this.elements.robotIP.value.trim();
        this.token = this.elements.token.value.trim();

        if (!this.robotIP ) {
            this.showNotification('请输入机器人IP地址和访问令牌', 'error');
            return;
        }

        this.updateConnectionStatus('connecting', '连接中...');
        this.elements.connectBtn.disabled = true;
        this.elements.connectText.innerHTML = '<span class="loading"></span> 连接中...';

        // 建立WebSocket连接
        const wsUrl = `ws://${window.location.host}/ws`;
        this.websocket = new WebSocket(wsUrl);

        this.websocket.onopen = () => {
            this.log('WebSocket连接已建立', 'info');
            
            // 发送连接请求
            this.websocket.send(JSON.stringify({
                action: 'connect',
                robot_ip: this.robotIP,
                token: this.token
            }));
        };

        this.websocket.onmessage = (event) => {
            try {
                const response = JSON.parse(event.data);
                this.handleWebSocketMessage(response);
            } catch (error) {
                this.log(`解析WebSocket消息失败: ${error.message}`, 'error');
            }
        };

        this.websocket.onerror = (error) => {
            this.log(`WebSocket错误: ${error}`, 'error');
            this.updateConnectionStatus('disconnected', '连接失败');
            this.resetConnectionUI();
        };

        this.websocket.onclose = () => {
            this.log('WebSocket连接已关闭', 'warning');
            this.updateConnectionStatus('disconnected', '连接已断开');
            this.resetConnectionUI();
        };
    }

    disconnect() {
        if (this.websocket) {
            this.websocket.close();
        }
        this.isConnected = false;
        this.updateConnectionStatus('disconnected', '已断开连接');
        this.resetConnectionUI();
        this.hideVideo();
        this.log('已断开与机器人的连接', 'info');
    }

    handleWebSocketMessage(response) {
        this.log(`收到消息: ${response.message}`, response.success ? 'success' : 'error');

        if (response.success) {
            switch (response.message) {
                case '连接成功':
                    this.isConnected = true;
                    this.updateConnectionStatus('connected', '已连接');
                    this.enableControlButtons();
                    this.showNotification('成功连接到机器人', 'success');
                    break;
                case '验证成功':
                    this.log('机器人验证成功', 'success');
                    break;
                case '命令发送成功':
                    this.showNotification('命令发送成功', 'success');
                    break;
                case '视频控制成功':
                    this.showNotification('视频控制成功', 'success');
                    break;
                case '视频帧':
                    this.handleVideoFrame(response.data);
                    break;
                case '收到消息':
                    this.log(`机器人消息: ${JSON.stringify(response.data)}`, 'info');
                    break;
            }
        } else {
            this.showNotification(`操作失败: ${response.message}`, 'error');
            if (response.message.includes('连接机器人失败')) {
                this.updateConnectionStatus('disconnected', '连接失败');
                this.resetConnectionUI();
            }
        }
    }

    handleVideoFrame(videoFrame) {
        // 处理视频帧数据
        // 这里可以根据需要处理视频帧
        this.log(`收到视频帧: 类型=${videoFrame.frame_type}, 时间戳=${videoFrame.timestamp}, 大小=${videoFrame.frame_data.length}字节`, 'info');
    }

    sendCommand() {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        const command = this.elements.commandSelect.value;
        if (!command) {
            this.showNotification('请选择要发送的命令', 'warning');
            return;
        }

        this.websocket.send(JSON.stringify({
            action: 'command',
            robot_ip: this.robotIP,
            token: this.token,
            command: command
        }));

        this.log(`发送命令: ${command}`, 'info');
    }

    sendQuickCommand(command) {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        this.websocket.send(JSON.stringify({
            action: 'command',
            robot_ip: this.robotIP,
            token: this.token,
            command: command
        }));

        this.log(`发送快速命令: ${command}`, 'info');
        this.showNotification(`已发送命令: ${command}`, 'success');
    }

    openVideo() {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        this.websocket.send(JSON.stringify({
            action: 'video',
            robot_ip: this.robotIP,
            token: this.token,
            command: 'open'
        }));

        this.log('开启视频流', 'info');
        this.showVideo();
    }

    closeVideo() {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        this.websocket.send(JSON.stringify({
            action: 'video',
            robot_ip: this.robotIP,
            token: this.token,
            command: 'close'
        }));

        this.log('关闭视频流', 'info');
        this.hideVideo();
    }

    showVideo() {
        this.elements.videoPlaceholder.style.display = 'none';
        this.elements.videoStream.style.display = 'block';
    }

    hideVideo() {
        this.elements.videoPlaceholder.style.display = 'flex';
        this.elements.videoStream.style.display = 'none';
    }

    updateConnectionStatus(status, text) {
        this.elements.connectionStatus.className = `connection-status status-${status}`;
        this.elements.statusText.textContent = text;
    }

    resetConnectionUI() {
        this.elements.connectBtn.disabled = false;
        this.elements.connectText.textContent = '连接机器人';
        this.disableControlButtons();
    }

    enableControlButtons() {
        this.elements.sendCommandBtn.disabled = false;
        this.elements.openVideoBtn.disabled = false;
        this.elements.closeVideoBtn.disabled = false;
        this.elements.quickCommands.forEach(btn => btn.disabled = false);
    }

    disableControlButtons() {
        this.elements.sendCommandBtn.disabled = true;
        this.elements.openVideoBtn.disabled = true;
        this.elements.closeVideoBtn.disabled = true;
        this.elements.quickCommands.forEach(btn => btn.disabled = true);
    }

    log(message, type = 'info') {
        const timestamp = new Date().toLocaleTimeString();
        const logEntry = document.createElement('div');
        logEntry.className = `log-entry log-${type}`;
        logEntry.textContent = `[${timestamp}] ${message}`;
        
        this.elements.logContent.appendChild(logEntry);
        this.elements.logContent.scrollTop = this.elements.logContent.scrollHeight;

        // 限制日志条目数量
        while (this.elements.logContent.children.length > 100) {
            this.elements.logContent.removeChild(this.elements.logContent.firstChild);
        }
    }

    showNotification(message, type = 'info') {
        const notification = this.elements.notification;
        notification.textContent = message;
        notification.className = `notification notification-${type}`;
        
        // 显示通知
        setTimeout(() => {
            notification.classList.add('show');
        }, 100);

        // 自动隐藏
        setTimeout(() => {
            notification.classList.remove('show');
        }, 3000);
    }
}

// 页面加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    window.robotController = new Go2RobotController();
    
    // 添加一些初始日志
    window.robotController.log('Go2 机器人控制面板已加载', 'success');
    window.robotController.log('请输入机器人IP地址和访问令牌，然后点击连接', 'info');
}); 