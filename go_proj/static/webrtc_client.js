// Go2 机器人 WebRTC 客户端
class Go2WebRTCClient {
    constructor() {
        this.robotIP = '';
        this.token = '';
        this.isConnected = false;
        this.peerConnection = null;
        this.clientID = null;
        this.reconnecting = false; // 新增重连状态
        
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

    async connect() {
        this.robotIP = this.elements.robotIP.value.trim();
        this.token = this.elements.token.value.trim();

        if (!this.robotIP) { // token不是必须的
            this.showNotification('请输入机器人IP地址和访问令牌', 'error');
            return;
        }

        this.updateConnectionStatus('connecting', '连接中...');
        this.elements.connectBtn.disabled = true;
        this.elements.connectText.innerHTML = '<span class="loading"></span> 连接中...';

        try {
            // 首先连接到机器人
            await this.connectToRobot();
            
            // 然后建立WebRTC连接
            await this.establishWebRTCConnection();
            
            // 自动开启视频
            await this.openVideo();
            
            this.isConnected = true;
            this.updateConnectionStatus('connected', '已连接');
            this.enableControlButtons();
            this.showNotification('成功连接到机器人，视频已自动开启', 'success');
            
        } catch (error) {
            this.log(`连接失败: ${error.message}`, 'error');
            this.updateConnectionStatus('disconnected', '连接失败');
            this.resetConnectionUI();
            this.showNotification(`连接失败: ${error.message}`, 'error');
        }
    }

    async connectToRobot() {
        const response = await fetch('/api/connect', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                robot_ip: this.robotIP,
                token: this.token
            })
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`连接机器人失败: ${errorText}`);
        }

        const result = await response.json();
        if (!result.success) {
            throw new Error(result.message);
        }

        this.log('机器人连接成功', 'success');
    }

    async establishWebRTCConnection() {
        this.log('开始建立WebRTC连接...', 'info');
        
        // 获取WebRTC客户端连接
        const clientUrl = `/webrtc/client?robot_ip=${encodeURIComponent(this.robotIP)}&token=${encodeURIComponent(this.token)}`;
        this.log(`请求WebRTC客户端: ${clientUrl}`, 'info');
        
        const clientResponse = await fetch(clientUrl);
        
        if (!clientResponse.ok) {
            const errorText = await clientResponse.text();
            this.log(`WebRTC客户端请求失败: ${clientResponse.status} - ${errorText}`, 'error');
            throw new Error(`获取WebRTC客户端失败: ${clientResponse.status} - ${errorText}`);
        }

        const clientResult = await clientResponse.json();
        this.log(`收到WebRTC客户端响应: ${JSON.stringify(clientResult)}`, 'info');
        
        this.clientID = clientResult.client_id || `client_${Date.now()}`;
        
        this.log(`WebRTC客户端ID: ${this.clientID}`, 'info');

        // 创建RTCPeerConnection - 本地局域网环境
        this.peerConnection = new RTCPeerConnection({
            // 本地局域网环境，不需要STUN服务器
            iceServers: [],
            iceTransportPolicy: 'all',
            bundlePolicy: 'max-bundle',
            rtcpMuxPolicy: 'require'
        });

        this.log('RTCPeerConnection创建成功（本地模式）', 'info');

        // 设置事件处理
        this.peerConnection.ontrack = (event) => {
            this.log('收到视频轨道', 'success');
            this.elements.videoStream.srcObject = event.streams[0];
            this.elements.videoPlaceholder.style.display = 'none';
            this.elements.videoStream.style.display = 'block';
            
            // 自动播放视频
            this.elements.videoStream.play().then(() => {
                this.log('视频开始播放', 'success');
            }).catch(e => {
                this.log(`视频播放失败: ${e.message}`, 'error');
            });
        };

        this.peerConnection.oniceconnectionstatechange = () => {
            this.log(`ICE连接状态: ${this.peerConnection.iceConnectionState}`, 'info');
            // 本地网络环境，ICE状态可能不同
            if (this.peerConnection.iceConnectionState === 'failed') {
                this.log('ICE连接失败，但继续处理视频流（本地网络）', 'warning');
            } else if (this.peerConnection.iceConnectionState === 'connected') {
                this.log('ICE连接成功', 'success');
            }
        };

        this.peerConnection.onconnectionstatechange = () => {
            this.log(`连接状态: ${this.peerConnection.connectionState}`, 'info');
            if (this.peerConnection.connectionState === 'connected') {
                this.log('WebRTC连接已建立，等待视频流...', 'success');
            } else if (this.peerConnection.connectionState === 'failed') {
                this.log('WebRTC连接失败', 'error');
                this.reconnectWebRTC();
            }
        };

        this.peerConnection.onicecandidate = (event) => {
            if (event.candidate) {
                this.log(`ICE候选地址: ${event.candidate.candidate}`, 'info');
            } else {
                this.log('ICE候选地址收集完成', 'info');
            }
        };

        // 设置远程描述（SDP提议）
        this.log('设置远程描述...', 'info');
        await this.peerConnection.setRemoteDescription(new RTCSessionDescription({
            type: clientResult.type,
            sdp: clientResult.sdp
        }));

        // 创建应答
        this.log('创建应答...', 'info');
        const answer = await this.peerConnection.createAnswer();
        await this.peerConnection.setLocalDescription(answer);

        // 发送应答到服务器
        this.log('发送应答到服务器...', 'info');
        const answerResponse = await fetch('/webrtc/answer', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                client_id: this.clientID,
                answer: {
                    type: answer.type,
                    sdp: answer.sdp
                }
            })
        });

        if (!answerResponse.ok) {
            const errorText = await answerResponse.text();
            this.log(`发送WebRTC应答失败: ${answerResponse.status} - ${errorText}`, 'error');
            throw new Error(`发送WebRTC应答失败: ${answerResponse.status} - ${errorText}`);
        }

        const answerResult = await answerResponse.json();
        this.log(`收到应答结果: ${JSON.stringify(answerResult)}`, 'info');
        
        if (!answerResult.success) {
            throw new Error(answerResult.message);
        }

        this.log('WebRTC连接建立成功', 'success');
    }

    async reconnectWebRTC() {
        if (this.reconnecting) {
            return; // 防止重复重连
        }
        
        this.reconnecting = true;
        this.log('开始重连WebRTC...', 'info');
        
        try {
            // 关闭现有连接
            if (this.peerConnection) {
                this.peerConnection.close();
                this.peerConnection = null;
            }
            
            // 等待一段时间后重连
            await new Promise(resolve => setTimeout(resolve, 2000));
            
            // 重新建立WebRTC连接
            await this.establishWebRTCConnection();
            
            this.log('WebRTC重连成功', 'success');
            
        } catch (error) {
            this.log(`WebRTC重连失败: ${error.message}`, 'error');
        } finally {
            this.reconnecting = false;
        }
    }

    disconnect() {
        if (this.peerConnection) {
            this.peerConnection.close();
            this.peerConnection = null;
        }
        
        this.isConnected = false;
        this.updateConnectionStatus('disconnected', '已断开连接');
        this.resetConnectionUI();
        this.hideVideo();
        this.log('已断开与机器人的连接', 'info');
    }

    async sendCommand() {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        const command = this.elements.commandSelect.value;
        if (!command) {
            this.showNotification('请选择要发送的命令', 'warning');
            return;
        }

        try {
            const response = await fetch('/api/command', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    robot_ip: this.robotIP,
                    token: this.token,
                    command: command
                })
            });

            if (!response.ok) {
                throw new Error('发送命令失败');
            }

            const result = await response.json();
            if (result.success) {
                this.log(`发送命令: ${command}`, 'success');
                this.showNotification('命令发送成功', 'success');
            } else {
                throw new Error(result.message);
            }
        } catch (error) {
            this.log(`发送命令失败: ${error.message}`, 'error');
            this.showNotification(`发送命令失败: ${error.message}`, 'error');
        }
    }

    async sendQuickCommand(command) {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        try {
            const response = await fetch('/api/command', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    robot_ip: this.robotIP,
                    token: this.token,
                    command: command
                })
            });

            if (!response.ok) {
                throw new Error('发送命令失败');
            }

            const result = await response.json();
            if (result.success) {
                this.log(`发送快速命令: ${command}`, 'success');
                this.showNotification(`已发送命令: ${command}`, 'success');
            } else {
                throw new Error(result.message);
            }
        } catch (error) {
            this.log(`发送快速命令失败: ${error.message}`, 'error');
            this.showNotification(`发送命令失败: ${error.message}`, 'error');
        }
    }

    async openVideo() {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        try {
            const response = await fetch('/api/video', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    robot_ip: this.robotIP,
                    token: this.token,
                    command: 'open'
                })
            });

            if (!response.ok) {
                throw new Error('开启视频失败');
            }

            const result = await response.json();
            if (result.success) {
                this.log('开启视频流', 'success');
                this.showNotification('视频已开启', 'success');
            } else {
                throw new Error(result.message);
            }
        } catch (error) {
            this.log(`开启视频失败: ${error.message}`, 'error');
            this.showNotification(`开启视频失败: ${error.message}`, 'error');
        }
    }

    async closeVideo() {
        if (!this.isConnected) {
            this.showNotification('请先连接到机器人', 'warning');
            return;
        }

        try {
            const response = await fetch('/api/video', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    robot_ip: this.robotIP,
                    token: this.token,
                    command: 'close'
                })
            });

            if (!response.ok) {
                throw new Error('关闭视频失败');
            }

            const result = await response.json();
            if (result.success) {
                this.log('关闭视频流', 'info');
                this.hideVideo();
                this.showNotification('视频已关闭', 'success');
            } else {
                throw new Error(result.message);
            }
        } catch (error) {
            this.log(`关闭视频失败: ${error.message}`, 'error');
            this.showNotification(`关闭视频失败: ${error.message}`, 'error');
        }
    }

    showVideo() {
        this.elements.videoPlaceholder.style.display = 'none';
        this.elements.videoStream.style.display = 'block';
        this.log('视频显示已开启', 'success');
        
        // 确保视频元素设置正确
        this.elements.videoStream.autoplay = true;
        this.elements.videoStream.muted = true;
        this.elements.videoStream.controls = true;
        this.elements.videoStream.playsInline = true;
    }

    hideVideo() {
        this.elements.videoPlaceholder.style.display = 'flex';
        this.elements.videoStream.style.display = 'none';
        this.log('视频显示已关闭', 'info');
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
    window.webrtcClient = new Go2WebRTCClient();
    
    // 添加一些初始日志
    window.webrtcClient.log('Go2 机器人 WebRTC 客户端已加载', 'success');
    window.webrtcClient.log('请输入机器人IP地址和访问令牌，然后点击连接', 'info');
}); 