# Copyright (c) 2024, RoboVerse community
#
# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions are met:
#
# 1. Redistributions of source code must retain the above copyright notice, this
#    list of conditions and the following disclaimer.
#
# 2. Redistributions in binary form must reproduce the above copyright notice,
#    this list of conditions and the following disclaimer in the documentation
#    and/or other materials provided with the distribution.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
# AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
# IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
# DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
# FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
# DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
# SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
# CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
# OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
# OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

import asyncio
import json
import os
import pygame
import math

from go2_webrtc import Go2Connection, ROBOT_CMD

# 虚拟手柄配置
JOY_SENSE = 0.2
WINDOW_WIDTH = 800
WINDOW_HEIGHT = 600
JOYSTICK_RADIUS = 80
BUTTON_RADIUS = 30

class VirtualJoystick:
    def __init__(self, x, y, radius, name):
        self.x = x
        self.y = y
        self.radius = radius
        self.name = name
        self.value_x = 0.0
        self.value_y = 0.0
        self.is_dragging = False
        self.drag_start = None
        
    def handle_event(self, event):
        if event.type == pygame.MOUSEBUTTONDOWN:
            if event.button == 1:  # 左键
                distance = math.sqrt((event.pos[0] - self.x)**2 + (event.pos[1] - self.y)**2)
                if distance <= self.radius:
                    self.is_dragging = True
                    self.drag_start = event.pos
                    
        elif event.type == pygame.MOUSEBUTTONUP:
            if event.button == 1:
                self.is_dragging = False
                self.value_x = 0.0
                self.value_y = 0.0
                
        elif event.type == pygame.MOUSEMOTION:
            if self.is_dragging:
                dx = event.pos[0] - self.x
                dy = event.pos[1] - self.y
                distance = math.sqrt(dx**2 + dy**2)
                
                if distance <= self.radius:
                    self.value_x = dx / self.radius
                    self.value_y = dy / self.radius
                else:
                    # 限制在圆形范围内
                    angle = math.atan2(dy, dx)
                    self.value_x = math.cos(angle)
                    self.value_y = math.sin(angle)
    
    def draw(self, screen):
        # 绘制摇杆背景
        pygame.draw.circle(screen, (100, 100, 100), (self.x, self.y), self.radius, 2)
        
        # 绘制摇杆当前位置
        stick_x = self.x + int(self.value_x * self.radius * 0.7)
        stick_y = self.y + int(self.value_y * self.radius * 0.7)
        pygame.draw.circle(screen, (255, 0, 0), (stick_x, stick_y), 15)
        
        # 绘制标签
        font = pygame.font.Font(None, 24)
        text = font.render(self.name, True, (255, 255, 255))
        screen.blit(text, (self.x - text.get_width()//2, self.y + self.radius + 10))

class VirtualButton:
    def __init__(self, x, y, radius, name, color=(0, 255, 0)):
        self.x = x
        self.y = y
        self.radius = radius
        self.name = name
        self.color = color
        self.is_pressed = False
        
    def handle_event(self, event):
        if event.type == pygame.MOUSEBUTTONDOWN:
            if event.button == 1:
                distance = math.sqrt((event.pos[0] - self.x)**2 + (event.pos[1] - self.y)**2)
                if distance <= self.radius:
                    self.is_pressed = True
                    
        elif event.type == pygame.MOUSEBUTTONUP:
            if event.button == 1:
                self.is_pressed = False
    
    def draw(self, screen):
        color = (255, 0, 0) if self.is_pressed else self.color
        pygame.draw.circle(screen, color, (self.x, self.y), self.radius)
        
        # 绘制标签
        font = pygame.font.Font(None, 20)
        text = font.render(self.name, True, (255, 255, 255))
        screen.blit(text, (self.x - text.get_width()//2, self.y + self.radius + 5))

class VirtualGamepad:
    def __init__(self):
        pygame.init()
        self.screen = pygame.display.set_mode((WINDOW_WIDTH, WINDOW_HEIGHT))
        pygame.display.set_caption("GO2 Virtual Joystick")
        
        # 创建虚拟摇杆
        self.left_stick = VirtualJoystick(200, 300, JOYSTICK_RADIUS, "Left")
        self.right_stick = VirtualJoystick(600, 300, JOYSTICK_RADIUS, "Right")
        
        # 创建虚拟按钮
        self.btn_a = VirtualButton(550, 150, BUTTON_RADIUS, "A", (0, 255, 0))
        self.btn_b = VirtualButton(600, 150, BUTTON_RADIUS, "B", (255, 255, 255))
        self.btn_x = VirtualButton(450, 150, BUTTON_RADIUS, "X", (0, 0, 255))
        self.btn_y = VirtualButton(500, 150, BUTTON_RADIUS, "Y", (255, 255, 0))
        
        self.clock = pygame.time.Clock()
        
    def handle_events(self):
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                return False
                
            # 处理摇杆事件
            self.left_stick.handle_event(event)
            self.right_stick.handle_event(event)
            
            # 处理按钮事件
            self.btn_a.handle_event(event)
            self.btn_b.handle_event(event)
            self.btn_x.handle_event(event)
            self.btn_y.handle_event(event)
            
        return True
    
    def draw(self):
        self.screen.fill((50, 50, 50))
        
        # 绘制摇杆
        self.left_stick.draw(self.screen)
        self.right_stick.draw(self.screen)
        
        # 绘制按钮
        self.btn_a.draw(self.screen)
        self.btn_b.draw(self.screen)
        self.btn_x.draw(self.screen)
        self.btn_y.draw(self.screen)
        
        # 绘制状态信息
        font = pygame.font.Font(None, 32)
        status_text = f"L Stick: ({self.left_stick.value_x:.2f}, {self.left_stick.value_y:.2f})"
        text = font.render(status_text, True, (255, 255, 255))
        self.screen.blit(text, (10, 10))
        
        status_text2 = f"R Stick: ({self.right_stick.value_x:.2f}, {self.right_stick.value_y:.2f})"
        text2 = font.render(status_text2, True, (255, 255, 255))
        self.screen.blit(text2, (10, 50))
        
        pygame.display.flip()
    
    def get_joystick_values(self):
        return {
            "Axis 0": self.left_stick.value_x,
            "Axis 1": self.left_stick.value_y,
            "Axis 2": self.right_stick.value_x,
            "Axis 3": self.right_stick.value_y,
            "a": 1 if self.btn_a.is_pressed else 0,
            "b": 1 if self.btn_b.is_pressed else 0,
            "x": 1 if self.btn_x.is_pressed else 0,
            "y": 1 if self.btn_y.is_pressed else 0,
        }
    
    def quit(self):
        pygame.quit()

def gen_command(cmd: int):
    command = {
        "type": "msg",
        "topic": "rt/api/sport/request",
        "data": {
            "header": {"identity": {"id": Go2Connection.generate_id(), "api_id": cmd}},
            "parameter": json.dumps(cmd),
        },
    }
    return json.dumps(command)

def gen_mov_command(x: float, y: float, z: float):
    x = x * JOY_SENSE
    y = y * JOY_SENSE

    command = {
        "type": "msg",
        "topic": "rt/api/sport/request",
        "data": {
            "header": {"identity": {"id": Go2Connection.generate_id(), "api_id": 1008}},
            "parameter": json.dumps({"x": x, "y": y, "z": z}),
        },
    }
    return json.dumps(command)

async def start_virtual_joy_bridge(robot_conn, virtual_pad):
    await robot_conn.connect_robot()

    while True:
        # 处理 pygame 事件
        if not virtual_pad.handle_events():
            break
            
        # 绘制界面
        virtual_pad.draw()
        
        # 获取虚拟手柄值
        joystick_values = virtual_pad.get_joystick_values()
        
        joy_move_x = joystick_values["Axis 1"]
        joy_move_y = joystick_values["Axis 0"]
        joy_move_z = joystick_values["Axis 2"]
        joy_btn_a_is_pressed = joystick_values["a"]
        joy_btn_b_is_pressed = joystick_values["b"]
        joy_btn_x_is_pressed = joystick_values["x"]
        joy_btn_y_is_pressed = joystick_values["y"]

        # 按钮控制
        if joy_btn_a_is_pressed == 1:
            robot_cmd = gen_command(ROBOT_CMD["StandUp"])
            robot_conn.data_channel.send(robot_cmd)

        if joy_btn_b_is_pressed == 1:
            robot_cmd = gen_command(ROBOT_CMD["StandDown"])
            robot_conn.data_channel.send(robot_cmd)
            
        if joy_btn_x_is_pressed == 1:
            robot_cmd = gen_command(ROBOT_CMD["StopMove"])
            robot_conn.data_channel.send(robot_cmd)
            
        if joy_btn_y_is_pressed == 1:
            robot_cmd = gen_command(ROBOT_CMD["Hello"])
            # robot_conn.data_channel.send(robot_cmd)

        # 摇杆移动控制
        if abs(joy_move_x) > 0.0 or abs(joy_move_y) > 0.0 or abs(joy_move_z) > 0.0:
            robot_cmd = gen_mov_command(joy_move_x, joy_move_y, joy_move_z)
            robot_conn.data_channel.send(robot_cmd)

        # 控制帧率
        virtual_pad.clock.tick(60)
        await asyncio.sleep(0.016)  # 约60FPS

async def main():
    # 创建虚拟手柄
    virtual_pad = VirtualGamepad()
    
    # 创建机器人连接
    conn = Go2Connection(
        "192.168.123.161",
        "",
    )

    try:
        await start_virtual_joy_bridge(conn, virtual_pad)
    except KeyboardInterrupt:
        print("程序被用户中断")
    finally:
        virtual_pad.quit()
        await conn.pc.close()

if __name__ == "__main__":
    asyncio.run(main()) 