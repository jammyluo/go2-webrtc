// Go2 机器人 WebRTC 客户端
class Go2WebRTCClient {
    constructor() {
        this.robotIP = '';
        this.token = '';
        this.isConnected = false;
        this.peerConnection = null;
        this.clientID = null;
        this.reconnecting = false; // 新增重连状态
        
        // 摇杆相关
        this.joysticks = {
            left: null,
            right: null
        };
        this.movementInterval = null;
        this.lastMovement = { x: 0, y: 0, z: 0 };
        
        this.initElements();
        this.initEventListeners();
        this.initJoysticks();
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
            quickCommands: document.querySelectorAll('.quick-command'),
            // 摇杆元素
            joystickLeft: document.getElementById('joystick-left'),
            joystickRight: document.getElementById('joystick-right'),
            // 测试按钮
            testStatusBtn: document.getElementById('test-status-btn')
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

        // 键盘控制
        document.addEventListener('keydown', (e) => this.handleKeyDown(e));
        document.addEventListener('keyup', (e) => this.handleKeyUp(e));

        // 测试状态更新按钮
        if (this.elements.testStatusBtn) {
            this.elements.testStatusBtn.addEventListener('click', () => {
                console.log('测试状态更新按钮被点击');
                console.log('当前连接状态:', this.isConnected);
                console.log('连接按钮显示状态:', this.elements.connectBtn.style.display);
                console.log('断开按钮显示状态:', this.elements.disconnectBtn.style.display);
                console.log('连接文本:', this.elements.connectText.textContent);
                console.log('状态文本:', this.elements.statusText.textContent);
                
                // 测试状态更新
                this.updateConnectionStatus('connected', '测试已连接');
                this.elements.connectBtn.style.display = 'none';
                this.elements.disconnectBtn.style.display = 'inline-block';
                
                setTimeout(() => {
                    this.updateConnectionStatus('disconnected', '测试未连接');
                    this.elements.connectBtn.style.display = 'inline-block';
                    this.elements.disconnectBtn.style.display = 'none';
                }, 2000);
            });
        }
    }

    // 初始化摇杆
    initJoysticks() {
        console.log('初始化摇杆...');
        
        // 检查元素是否存在
        if (!this.elements.joystickLeft || !this.elements.joystickRight) {
            console.error('摇杆元素未找到，等待DOM加载...');
            setTimeout(() => this.initJoysticks(), 100);
            return;
        }
        
        console.log('左摇杆元素:', this.elements.joystickLeft);
        console.log('右摇杆元素:', this.elements.joystickRight);
        
        // 设置摇杆
        this.setupSimpleJoystick(this.elements.joystickLeft, 'left');
        this.setupSimpleJoystick(this.elements.joystickRight, 'right');
        
        this.startMovementControl();
        console.log('摇杆初始化完成');
    }

    // 简单的摇杆设置
    setupSimpleJoystick(element, side) {
        console.log(`设置${side}摇杆:`, element);
        
        const knob = element.querySelector('.joystick-knob');
        const valueDisplay = element.querySelector('.joystick-value');
        
        if (!knob || !valueDisplay) {
            console.error(`${side}摇杆元素不完整`);
            return;
        }
        
        let isActive = false;
        let currentX = 0, currentY = 0;
        const maxDistance = 50;
        const self = this;
        
        // 更新位置函数
        const updatePosition = (clientX, clientY) => {
            const rect = element.getBoundingClientRect();
            const centerX = rect.left + rect.width / 2;
            const centerY = rect.top + rect.height / 2;
            
            const deltaX = clientX - centerX;
            const deltaY = clientY - centerY;
            const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
            
            if (distance > maxDistance) {
                const angle = Math.atan2(deltaY, deltaX);
                currentX = Math.cos(angle) * maxDistance;
                currentY = Math.sin(angle) * maxDistance;
            } else {
                currentX = deltaX;
                currentY = deltaY;
            }
            
            knob.style.transform = `translate(calc(-50% + ${currentX}px), calc(-50% + ${currentY}px))`;
            
            const normalizedX = (currentX / maxDistance).toFixed(1);
            const normalizedY = (currentY / maxDistance).toFixed(1);
            
            if (side === 'left') {
                valueDisplay.textContent = `前后: ${normalizedX} 转向: ${normalizedY}`;
            } else {
                valueDisplay.textContent = `左右: ${normalizedX} 上下: ${normalizedY}`;
            }
            
            self.joysticks[side] = {
                x: parseFloat(normalizedX),
                y: parseFloat(normalizedY),
                active: true
            };
            
            console.log(`${side}摇杆: X=${normalizedX}, Y=${normalizedY}`);
            console.log(`${side}摇杆数据:`, self.joysticks[side]);
        };
        
        // 重置位置函数
        const resetPosition = () => {
            knob.style.transform = 'translate(-50%, -50%)';
            currentX = 0;
            currentY = 0;
            
            if (side === 'left') {
                valueDisplay.textContent = '前后: 0.0 转向: 0.0';
            } else {
                valueDisplay.textContent = '左右: 0.0 上下: 0.0';
            }
            
            if (self.joysticks[side]) {
                self.joysticks[side].active = false;
                console.log(`${side}摇杆重置，active设为false`);
            }
            
            console.log(`${side}摇杆重置`);
        };
        
        // 鼠标事件
        element.addEventListener('mousedown', (e) => {
            console.log(`${side}摇杆按下`);
            e.preventDefault();
            isActive = true;
            element.classList.add('active');
            updatePosition(e.clientX, e.clientY);
        });
        
        document.addEventListener('mousemove', (e) => {
            if (isActive) {
                e.preventDefault();
                updatePosition(e.clientX, e.clientY);
            }
        });
        
        document.addEventListener('mouseup', () => {
            if (isActive) {
                console.log(`${side}摇杆释放`);
                isActive = false;
                element.classList.remove('active');
                resetPosition();
            }
        });
        
        // 触摸事件
        element.addEventListener('touchstart', (e) => {
            console.log(`${side}摇杆触摸开始`);
            e.preventDefault();
            isActive = true;
            element.classList.add('active');
            const touch = e.touches[0];
            updatePosition(touch.clientX, touch.clientY);
        });
        
        document.addEventListener('touchmove', (e) => {
            if (isActive) {
                e.preventDefault();
                const touch = e.touches[0];
                updatePosition(touch.clientX, touch.clientY);
            }
        });
        
        document.addEventListener('touchend', () => {
            if (isActive) {
                console.log(`${side}摇杆触摸结束`);
                isActive = false;
                element.classList.remove('active');
                resetPosition();
            }
        });
        
        console.log(`${side}摇杆设置完成`);
    }

    // 启动移动控制循环
    startMovementControl() {
        this.movementInterval = setInterval(() => {
            this.updateMovement();
        }, 100); // 10Hz更新频率
    }

    // 更新移动控制
    updateMovement() {
        console.log('updateMovement 被调用, isConnected:', this.isConnected);
        
        // 即使未连接也允许摇杆操作，用于测试
        let x = 0, y = 0, z = 0;

        // 左摇杆：前后移动(X)和转向(Z)
        if (this.joysticks.left && this.joysticks.left.active) {
            x = this.joysticks.left.y * -1;
            z = this.joysticks.left.x * -1;
            console.log('左摇杆激活:', { x, z });
        }

        // 右摇杆：左右移动(Y)和上下移动(暂未使用)
        if (this.joysticks.right && this.joysticks.right.active) {
            y = this.joysticks.right.x * -1;
            console.log('右摇杆激活:', { y });
            // 上下移动暂时不使用，可以后续扩展
        }

        // 应用死区
        const deadzone = 0.1;
        x = Math.abs(x) > deadzone ? x : 0;
        y = Math.abs(y) > deadzone ? y : 0;
        z = Math.abs(z) > deadzone ? z : 0;

        console.log('处理后的移动值:', { x, y, z });
        console.log('上次移动值:', this.lastMovement);

        // 检查是否有变化
        if (x !== this.lastMovement.x || y !== this.lastMovement.y || z !== this.lastMovement.z) {
            console.log('移动值发生变化，准备发送命令');
            this.lastMovement = { x, y, z };
            
            if (this.isConnected) {
                this.sendMovementCommand(x, y, z);
            } else {
                console.log('未连接到机器人，跳过发送移动命令');
            }
        } else {
            console.log('移动值无变化，跳过发送命令');
        }
    }

    // 发送移动命令
    async sendMovementCommand(x, y, z) {
        console.log('sendMovementCommand 被调用:', { x, y, z });
        
        if (!this.isConnected) {
            console.log('未连接到机器人，跳过发送移动命令');
            return;
        }

        try {
            console.log('准备发送移动命令到API...');
            const response = await fetch('/api/command', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    robot_ip: this.robotIP,
                    token: this.token,
                    command: 'Move',
                    data: { x, y, z }
                })
            });

            console.log('API响应状态:', response.status);

            if (!response.ok) {
                throw new Error('发送移动命令失败');
            }

            const result = await response.json();
            console.log('API响应结果:', result);
            
            if (!result.success) {
                throw new Error(result.message);
            }
            
            console.log('移动命令发送成功');
        } catch (error) {
            console.error('移动命令发送失败:', error);
            this.log(`移动命令发送失败: ${error.message}`, 'error');
        }
    }

    // 键盘控制处理
    handleKeyDown(event) {
        if (!this.isConnected) return;

        const key = event.key.toLowerCase();
        let x = 0, y = 0, z = 0;

        switch (key) {
            case 'w': // 前进
                x = 0.8;
                break;
            case 's': // 后退
                x = -0.4;
                break;
            case 'a': // 左移
                y = 0.4;
                break;
            case 'd': // 右移
                y = -0.4;
                break;
            case 'q': // 左转
                z = 2;
                break;
            case 'e': // 右转
                z = -2;
                break;
            default:
                return;
        }

        this.sendMovementCommand(x, y, z);
    }

    handleKeyUp(event) {
        if (!this.isConnected) return;

        const key = event.key.toLowerCase();
        if (['w', 's', 'a', 'd', 'q', 'e'].includes(key)) {
            // 停止移动
            this.sendMovementCommand(0, 0, 0);
        }
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

        console.log('开始连接流程...');
        this.updateConnectionStatus('connecting', '连接中...');
        this.elements.connectBtn.disabled = true;
        this.elements.connectText.innerHTML = '<span class="loading"></span> 连接中...';

        try {
            console.log('步骤1: 连接到机器人...');
            // 首先连接到机器人
            await this.connectToRobot();
            
            console.log('步骤2: 建立WebRTC连接...');
            // 然后建立WebRTC连接
            await this.establishWebRTCConnection();
            
            console.log('连接成功，更新状态...');
            this.isConnected = true;
            this.updateConnectionStatus('connected', '已连接');
            
            // 更新连接按钮状态
            console.log('更新按钮状态...');
            this.elements.connectBtn.style.display = 'none';
            this.elements.disconnectBtn.style.display = 'inline-block';
            this.elements.connectText.textContent = '连接机器人';
            
            this.enableControlButtons();
            
            console.log('步骤3: 开启视频...');
            // 自动开启视频
            await this.openVideo();
            
            this.showNotification('成功连接到机器人，视频已自动开启', 'success');
            console.log('连接流程完成');
            
        } catch (error) {
            console.error('连接失败:', error);
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
        const command = this.elements.commandSelect.value;
        if (!command) {
            this.showNotification('请选择要发送的命令', 'warning');
            return;
        }
        if (!this.isConnected && command != "Shoot") {
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
        if (!this.isConnected && command != "Shoot") {
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
        // 在连接过程中也允许调用
        if (!this.isConnected && !this.robotIP) {
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
        console.log('更新连接状态:', status, text);
        this.elements.connectionStatus.className = `connection-status status-${status}`;
        this.elements.statusText.textContent = text;
        console.log('连接状态元素:', this.elements.connectionStatus);
        console.log('状态文本元素:', this.elements.statusText);
    }

    resetConnectionUI() {
        this.elements.connectBtn.disabled = false;
        this.elements.connectBtn.style.display = 'inline-block';
        this.elements.disconnectBtn.style.display = 'none';
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
    
    // 测试摇杆元素是否存在
    setTimeout(() => {
        const leftJoystick = document.getElementById('joystick-left');
        const rightJoystick = document.getElementById('joystick-right');
        
        if (leftJoystick) {
            console.log('左摇杆元素存在:', leftJoystick);
            const knob = leftJoystick.querySelector('.joystick-knob');
            const value = leftJoystick.querySelector('.joystick-value');
            console.log('左摇杆按钮:', knob);
            console.log('左摇杆数值显示:', value);
        } else {
            console.error('左摇杆元素不存在');
        }
        
        if (rightJoystick) {
            console.log('右摇杆元素存在:', rightJoystick);
            const knob = rightJoystick.querySelector('.joystick-knob');
            const value = rightJoystick.querySelector('.joystick-value');
            console.log('右摇杆按钮:', knob);
            console.log('右摇杆数值显示:', value);
        } else {
            console.error('右摇杆元素不存在');
        }
    }, 1000);
});