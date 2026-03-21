# 前端访问故障排查

## 当前状态

✅ **前端服务运行正常**
- 端口：8001
- HTTP状态：200 OK
- Node.js：v18.20.8
- 监听状态：tcp46 *.8001 LISTEN

## 浏览器显示"连接被拒绝"的原因

### 1. 浏览器缓存问题
最常见原因是浏览器缓存了之前的连接失败状态。

**解决方案**：
1. **硬刷新页面**
   - Mac: `Command + Shift + R`
   - Windows/Linux: `Ctrl + Shift + R`

2. **清空缓存并硬性重新加载**
   - 打开开发者工具（F12）
   - 右键点击刷新按钮
   - 选择"清空缓存并硬性重新加载"

3. **使用无痕/隐私窗口**
   - Mac: `Command + Shift + N`
   - Windows/Linux: `Ctrl + Shift + N`
   - 访问 http://localhost:8001

### 2. 端口混淆
用户可能访问了错误的端口。

**正确地址**：
- 前端：http://localhost:8001
- 后端：http://localhost:8000

**直接访问登录页**：
http://localhost:8001/#/login

### 3. 之前的服务未完全关闭
之前的失败尝试可能留下了僵尸进程。

**验证服务**：
```bash
# 检查端口监听
lsof -i:8001

# 检查进程
ps aux | grep "umi dev"

# 测试连接
curl http://localhost:8001
```

### 4. 浏览器安全策略
某些浏览器安全设置可能阻止本地连接。

**尝试**：
- 使用 Chrome/Firefox
- 禁用浏览器扩展
- 检查浏览器安全设置

### 5. 网络或防火墙问题
虽然是本地连接，但某些安全软件可能拦截。

**检查**：
- 暂时禁用防火墙
- 检查安全软件设置
- 确认 localhost/127.0.0.1 未被hosts文件重定向

## 验证步骤

### 1. 终端测试
```bash
# 测试HTTP连接
curl -v http://localhost:8001

# 应该返回 200 OK 和 HTML 内容
```

### 2. 查看服务日志
```bash
cd web
# 检查是否有错误日志
# 正常应该看到 "App listening at: http://localhost:8001"
```

### 3. 检查网络监听
```bash
netstat -an | grep 8001
# 应该看到 *.8001 LISTEN
```

## 当前验证结果

✅ curl 测试：200 OK
✅ 端口监听：正常
✅ HTML 返回：正确
✅ 服务进程：运行中

**结论**：服务端完全正常，问题在浏览器端。

## 推荐操作顺序

1. **在新的无痕窗口打开**
   http://localhost:8001/#/login
   
2. **如果还是不行，完全关闭浏览器后重新打开**

3. **尝试其他浏览器**

4. **直接访问 IP 地址**
   http://127.0.0.1:8001

5. **检查 hosts 文件**
   ```bash
   cat /etc/hosts | grep localhost
   # 应该是: 127.0.0.1 localhost
   ```

## 开发者工具检查

打开浏览器开发者工具（F12）查看：

1. **Console 标签**
   - 是否有 JavaScript 错误
   - 是否有网络请求失败

2. **Network 标签**
   - 检查 http://localhost:8001 请求
   - 查看状态码和响应

3. **Application 标签**
   - 清除所有存储数据
   - 清除所有 Cookies

## 最终确认

如果以上步骤都完成，但问题仍存在，请提供：
1. 浏览器控制台的完整错误信息
2. Network 标签中的请求详情
3. 使用的浏览器和版本
4. `curl http://localhost:8001` 的完整输出

---

## 快速解决（99%有效）

**关闭所有浏览器窗口 → 重新打开 → 在无痕窗口访问：**
http://localhost:8001/#/login
