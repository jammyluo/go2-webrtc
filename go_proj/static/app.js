// Go2 机器人控制应用
class Go2RobotController {
    constructor() {
        this.websocket = null;
        this.isConnected = false;
        this.robotIP = '';
        this.token = '';
        
        // 视频处理相关
        this.mediaSource = null;
        this.sourceBuffer = null;
        
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
        this.cleanupVideoResources(); // 清理视频资源
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
        try {
            // 解析frame_type中的RTP信息
            let rtpInfo = {};
            if (videoFrame.frame_type && videoFrame.frame_type.startsWith('{')) {
                try {
                    rtpInfo = JSON.parse(videoFrame.frame_type);
                } catch (e) {
                    this.log(`解析RTP信息失败: ${e.message}`, 'error');
                }
            }

            // 记录详细的视频帧信息
            this.log(`收到视频帧: 大小=${videoFrame.frame_size}字节, 时间戳=${videoFrame.timestamp}`, 'info');
            
            if (Object.keys(rtpInfo).length > 0) {
                this.log(`RTP信息: 序列号=${rtpInfo.sequence}, SSRC=${rtpInfo.ssrc}, 负载类型=${rtpInfo.payload_type}, 标记=${rtpInfo.marker}`, 'info');
            }

            // 尝试解码和显示视频
            this.processVideoData(videoFrame.frame_data, rtpInfo);
            
        } catch (error) {
            this.log(`处理视频帧失败: ${error.message}`, 'error');
        }
    }

    processVideoData(base64Data, rtpInfo) {
        try {
            // 解码base64数据
            const binaryData = atob(base64Data);
            const bytes = new Uint8Array(binaryData.length);
            for (let i = 0; i < binaryData.length; i++) {
                bytes[i] = binaryData.charCodeAt(i);
            }

            // 根据负载类型处理不同的视频编码
            const payloadType = rtpInfo.payload_type || 96; // 默认H.264
            
            if (payloadType === 96 || payloadType === 97) {
                // H.264编码
                this.processH264Video(bytes, rtpInfo);
            } else if (payloadType === 98 || payloadType === 99) {
                // VP8/VP9编码
                this.processVP8Video(bytes, rtpInfo);
            } else {
                // 其他编码格式
                this.log(`不支持的视频编码格式: ${payloadType}`, 'warning');
                // 尝试通用处理
                this.processGenericVideo(bytes, rtpInfo);
            }

        } catch (error) {
            this.log(`解码视频数据失败: ${error.message}`, 'error');
        }
    }

    processH264Video(bytes, rtpInfo) {
        // 检查是否是关键帧
        const isKeyFrame = rtpInfo.marker || this.isH264KeyFrame(bytes);
        
        if (isKeyFrame) {
            this.log('检测到H.264关键帧', 'success');
        }

        // 使用MediaSource API处理实时视频流
        this.processVideoWithMediaSource(bytes, 'video/mp4; codecs="avc1.42E01E"', rtpInfo);
    }

    processVP8Video(bytes, rtpInfo) {
        // VP8视频处理
        this.processVideoWithMediaSource(bytes, 'video/webm; codecs="vp8"', rtpInfo);
    }

    processGenericVideo(bytes, rtpInfo) {
        // 通用视频处理
        this.processVideoWithMediaSource(bytes, 'video/mp4', rtpInfo);
    }

    processVideoWithMediaSource(bytes, mimeType, rtpInfo) {
        try {
            // 检查是否支持MediaSource
            if (!window.MediaSource) {
                this.log('浏览器不支持MediaSource API，使用备用方案', 'warning');
                this.processVideoWithBlob(bytes, mimeType);
                return;
            }

            // 获取或创建MediaSource
            if (!this.mediaSource) {
                this.mediaSource = new MediaSource();
                this.mediaSource.addEventListener('sourceopen', () => {
                    this.log('MediaSource已打开', 'info');
                    this.sourceBuffer = this.mediaSource.addSourceBuffer(mimeType);
                });
            }

            // 如果sourceBuffer可用，添加数据
            if (this.sourceBuffer && !this.sourceBuffer.updating) {
                try {
                    this.sourceBuffer.appendBuffer(bytes);
                    this.log(`添加视频数据: ${bytes.length}字节`, 'info');
                } catch (e) {
                    this.log(`添加视频数据失败: ${e.message}`, 'error');
                    // 如果失败，回退到Blob方案
                    this.processVideoWithBlob(bytes, mimeType);
                }
            } else {
                // 如果sourceBuffer不可用，使用Blob方案
                this.processVideoWithBlob(bytes, mimeType);
            }

        } catch (error) {
            this.log(`MediaSource处理失败: ${error.message}`, 'error');
            // 回退到Blob方案
            this.processVideoWithBlob(bytes, mimeType);
        }
    }

    processVideoWithBlob(bytes, mimeType) {
        try {
            // 创建Blob URL用于视频显示
            const blob = new Blob([bytes], { type: mimeType });
            const videoUrl = URL.createObjectURL(blob);
            
            // 更新视频元素
            const videoElement = this.elements.videoStream;
            if (videoElement.tagName === 'VIDEO') {
                // 如果之前有URL，释放它
                if (videoElement.src && videoElement.src.startsWith('blob:')) {
                    URL.revokeObjectURL(videoElement.src);
                }
                
                videoElement.src = videoUrl;
                videoElement.style.display = 'block';
                
                videoElement.play().catch(e => {
                    this.log(`播放视频失败: ${e.message}`, 'error');
                });
            } else {
                // 如果videoStream不是video元素，创建一个
                this.createVideoElement(videoUrl);
            }
        } catch (error) {
            this.log(`Blob视频处理失败: ${error.message}`, 'error');
        }
    }

    isH264KeyFrame(bytes) {
        // 简单的H.264关键帧检测
        // 查找NAL单元类型5 (IDR帧)
        for (let i = 0; i < bytes.length - 3; i++) {
            if (bytes[i] === 0 && bytes[i + 1] === 0 && bytes[i + 2] === 0 && bytes[i + 3] === 1) {
                const nalType = bytes[i + 4] & 0x1F;
                if (nalType === 5) {
                    return true;
                }
            }
        }
        return false;
    }

    createVideoElement(videoUrl) {
        // 创建video元素
        const videoElement = document.createElement('video');
        videoElement.autoplay = true;
        videoElement.controls = true;
        videoElement.muted = true; // 静音以避免自动播放问题
        videoElement.style.width = '100%';
        videoElement.style.height = '100%';
        videoElement.style.borderRadius = '8px';
        videoElement.style.backgroundColor = '#000';
        videoElement.src = videoUrl;
        
        // 添加错误处理
        videoElement.addEventListener('error', (e) => {
            this.log(`视频播放错误: ${e.message}`, 'error');
        });
        
        videoElement.addEventListener('loadstart', () => {
            this.log('视频开始加载', 'info');
        });
        
        videoElement.addEventListener('canplay', () => {
            this.log('视频可以播放', 'success');
            this.elements.videoPlaceholder.style.display = 'none';
        });
        
        // 替换现有的视频容器
        this.elements.videoStream.innerHTML = '';
        this.elements.videoStream.appendChild(videoElement);
        
        // 尝试播放
        videoElement.play().catch(e => {
            this.log(`播放视频失败: ${e.message}`, 'error');
            // 显示错误信息
            this.elements.videoStream.innerHTML = `
                <div style="text-align: center; padding: 20px; color: #666;">
                    <div style="font-size: 48px; margin-bottom: 10px;">⚠️</div>
                    <div>视频播放失败</div>
                    <div style="font-size: 12px; margin-top: 10px;">${e.message}</div>
                </div>
            `;
        });
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
        this.log('视频显示已开启', 'success');
    }

    hideVideo() {
        this.elements.videoPlaceholder.style.display = 'flex';
        this.elements.videoStream.style.display = 'none';
        this.cleanupVideoResources();
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

    cleanupVideoResources() {
        if (this.mediaSource) {
            this.mediaSource.removeSourceBuffer(this.sourceBuffer);
            this.sourceBuffer = null;
        }
        if (this.elements.videoStream.src && this.elements.videoStream.src.startsWith('blob:')) {
            URL.revokeObjectURL(this.elements.videoStream.src);
            this.elements.videoStream.src = ''; // 清除src属性
        }
    }
}

// 页面加载完成后初始化应用
document.addEventListener('DOMContentLoaded', () => {
    window.robotController = new Go2RobotController();
    
    // 添加一些初始日志
    window.robotController.log('Go2 机器人控制面板已加载', 'success');
    window.robotController.log('请输入机器人IP地址和访问令牌，然后点击连接', 'info');
}); 