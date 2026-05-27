const express = require('express');
const cors = require('cors');
const { exec } = require('child_process');
const util = require('util');
const jwt = require('jsonwebtoken');
const execPromise = util.promisify(exec);

const app = express();
app.use(cors());
app.use(express.json());

const SECRET_KEY = 'book-trace-secret-key-2025';

// 颜色定义（终端彩色输出）
const colors = {
    reset: '\x1b[0m',
    bright: '\x1b[1m',
    red: '\x1b[31m',
    green: '\x1b[32m',
    yellow: '\x1b[33m',
    blue: '\x1b[34m',
    magenta: '\x1b[35m',
    cyan: '\x1b[36m'
};

function log(color, prefix, msg) {
    console.log(`${color}[${prefix}]${colors.reset} ${msg}`);
}

const users = {
    'publisher': { id: 1, name: '机械工业出版社', role: 'publisher', roleName: '出版社', password: '123' },
    'printer': { id: 2, name: '新华印刷厂', role: 'printer', roleName: '印刷厂', password: '123' },
    'wholesaler': { id: 3, name: '新华批发商', role: 'wholesaler', roleName: '批发商', password: '123' },
    'bookstore': { id: 4, name: '新华书店', role: 'bookstore', roleName: '书店', password: '123' },
    'regulator': { id: 5, name: '新闻出版署', role: 'regulator', roleName: '监管机构', password: '123' },
    'user': { id: 6, name: '普通用户', role: 'user', roleName: '普通用户', password: '123' }
};

let purchaseRecords = [];

const fabricEnv = {
    PATH: process.env.PATH,
    FABRIC_CFG_PATH: '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/config',
    CORE_PEER_TLS_ENABLED: 'true',
    CORE_PEER_LOCALMSPID: 'Org1MSP',
    CORE_PEER_TLS_ROOTCERT_FILE: '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt',
    CORE_PEER_MSPCONFIGPATH: '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp',
    CORE_PEER_ADDRESS: 'peer0.org1.example.com:7051'
};

const ordererCA = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem';
const peer0CA = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt';
const peer1CA = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt';
const baseDir = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network';

async function queryChaincode(func, args) {
    log(colors.cyan, 'QUERY', `查询链码: ${func}`);
    const command = `cd ${baseDir} && peer chaincode query -C book-trace-channel -n booktrace -c '{"Args":["${func}","${args.join('","')}"]}' --tls --cafile ${ordererCA}`;
    const { stdout } = await execPromise(command, { env: fabricEnv });
    let result = stdout.trim();
    if (result.includes('Query result: ')) result = result.split('Query result: ')[1];
    return JSON.parse(result);
}

async function invokeChaincode(func, args) {
    log(colors.magenta, 'INVOKE', `调用链码: ${func}`);
    log(colors.magenta, 'INVOKE', `参数: ${JSON.stringify(args)}`);
    const command = `cd ${baseDir} && peer chaincode invoke -C book-trace-channel -n booktrace -c '{"Args":["${func}","${args.join('","')}"]}' --tls --cafile ${ordererCA} --peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles ${peer0CA} --peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles ${peer1CA} --waitForEvent`;
    const { stdout } = await execPromise(command, { env: fabricEnv });
    log(colors.green, 'INVOKE', `成功: ${func}`);
    return stdout;
}

function checkRole(allowedRoles) {
    return (req, res, next) => {
        const token = req.headers.authorization?.split(' ')[1];
        if (!token) return res.status(401).json({ error: '未登录' });
        try {
            const decoded = jwt.verify(token, SECRET_KEY);
            if (!allowedRoles.includes(decoded.role)) return res.status(403).json({ error: '权限不足' });
            req.user = decoded;
            next();
        } catch (error) {
            res.status(401).json({ error: '登录已过期' });
        }
    };
}

// ==================== API 路由 ====================

app.post('/api/login', (req, res) => {
    const { username, password } = req.body;
    log(colors.blue, 'LOGIN', `用户: ${username}`);
    const user = users[username];
    if (user && user.password === password) {
        const token = jwt.sign({ id: user.id, name: user.name, role: user.role, roleName: user.roleName }, SECRET_KEY, { expiresIn: '8h' });
        log(colors.green, 'LOGIN', `成功: ${username}`);
        res.json({ success: true, token, user: { name: user.name, role: user.role, roleName: user.roleName } });
    } else {
        log(colors.red, 'LOGIN', `失败: ${username}`);
        res.status(401).json({ error: '用户名或密码错误' });
    }
});

app.post('/api/register', (req, res) => {
    const { username, password, role } = req.body;
    log(colors.blue, 'REGISTER', `新用户: ${username}, 角色: ${role}`);
    if (users[username]) return res.status(400).json({ error: '用户名已存在' });
    const validRoles = ['publisher', 'printer', 'wholesaler', 'bookstore', 'regulator', 'user'];
    const userRole = validRoles.includes(role) ? role : 'user';
    const roleNames = { 'publisher': '出版社', 'printer': '印刷厂', 'wholesaler': '批发商', 'bookstore': '书店', 'regulator': '监管机构', 'user': '普通用户' };
    const newId = Object.keys(users).length + 1;
    users[username] = { id: newId, name: username, role: userRole, roleName: roleNames[userRole], password: password };
    const token = jwt.sign({ id: newId, name: username, role: userRole, roleName: roleNames[userRole] }, SECRET_KEY, { expiresIn: '8h' });
    log(colors.green, 'REGISTER', `成功: ${username}`);
    res.json({ success: true, token, user: { name: username, role: userRole, roleName: roleNames[userRole] }, message: '注册成功' });
});

app.get('/api/books', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore', 'regulator', 'user']), async (req, res) => {
    log(colors.cyan, 'QUERY', `用户 ${req.user.name} 查询图书列表`);
    try {
        const books = await queryChaincode('QueryAllBooks', []);
        res.json(books);
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.get('/api/books/:isbn', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore', 'regulator', 'user']), async (req, res) => {
    log(colors.cyan, 'QUERY', `用户 ${req.user.name} 查询图书: ${req.params.isbn}`);
    try {
        const { isbn } = req.params;
        const book = await queryChaincode('QueryBookByISBN', [isbn]);
        res.json(book);
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.post('/api/books', checkRole(['publisher']), async (req, res) => {
    const { isbn, bookName, author, publisher, publishDate, category, price, quantity } = req.body;
    log(colors.yellow, 'CREATE', `用户 ${req.user.name} 创建图书: ${isbn} - ${bookName}, 数量: ${quantity || 1}`);
    try {
        await invokeChaincode('CreateBook', [
            isbn, bookName, author, publisher, publishDate, category, price.toString(), 
            (quantity || 1).toString(), `batch_${Date.now()}`, '普通纸', '100', '0.5'
        ]);
        log(colors.green, 'CREATE', `成功: ${isbn}`);
        res.json({ success: true, message: '图书创建成功' });
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.put('/api/books/:isbn/state', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore']), async (req, res) => {
    const { isbn } = req.params;
    const { newState, remark } = req.body;
    log(colors.yellow, 'UPDATE', `用户 ${req.user.name} 更新图书状态: ${isbn} -> ${newState}`);
    try {
        await invokeChaincode('UpdateBookState', [isbn, newState, req.user.roleName, remark || '', `batch_${Date.now()}`, '0', '公路']);
        log(colors.green, 'UPDATE', `成功: ${isbn} -> ${newState}`);
        res.json({ success: true, message: '状态更新成功' });
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.post('/api/verify', async (req, res) => {
    const { isbn, ip, location, userAgent } = req.body;
    log(colors.cyan, 'VERIFY', `防伪码验证: ${isbn}, IP: ${ip}`);
    try {
        const result = await queryChaincode('VerifyAntiCounterfeit', [isbn, ip || 'unknown', location || 'unknown', userAgent || 'unknown']);
        log(colors.green, 'VERIFY', `结果: ${result.message}`);
        res.json(result);
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.post('/api/books/:isbn/buy', checkRole(['user']), async (req, res) => {
    const { isbn } = req.params;
    const { count } = req.body;
    const buyCount = count || 1;
    log(colors.yellow, 'BUY', `用户 ${req.user.name} 购买: ${isbn} x ${buyCount}`);
    try {
        const book = await queryChaincode('QueryBookByISBN', [isbn]);
        if (book.currentState !== '书店上架') return res.status(400).json({ error: '该书当前不可购买' });
        if (book.quantity < buyCount) return res.status(400).json({ error: `库存不足，当前库存: ${book.quantity}` });
        await invokeChaincode('BuyBook', [isbn, req.user.id.toString(), req.user.name, buyCount.toString()]);
        for (let i = 0; i < buyCount; i++) {
            purchaseRecords.push({ id: Date.now() + i, isbn: book.isbn, bookName: book.bookName, buyer: req.user.name, buyTime: new Date().toISOString(), price: book.price });
        }
        log(colors.green, 'BUY', `成功: ${isbn} x ${buyCount}`);
        res.json({ success: true, message: `购买成功！共 ${buyCount} 本` });
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.get('/api/my-purchases', checkRole(['user']), async (req, res) => {
    log(colors.cyan, 'QUERY', `用户 ${req.user.name} 查询购买记录`);
    res.json(purchaseRecords.filter(p => p.buyer === req.user.name));
});

app.post('/api/books/:isbn/review', checkRole(['user']), async (req, res) => {
    const { isbn } = req.params;
    const { rating, reviewType, content } = req.body;
    log(colors.yellow, 'REVIEW', `用户 ${req.user.name} 评价图书: ${isbn}, 类型: ${reviewType}`);
    try {
        const hasBought = purchaseRecords.some(p => p.buyer === req.user.name && p.isbn === isbn);
        if (!hasBought) return res.status(403).json({ error: '只有购买过该书的用户才能评价' });
        await invokeChaincode('AddReview', [isbn, req.user.id.toString(), req.user.name, rating.toString(), reviewType, content, hasBought ? 'true' : 'false']);
        log(colors.green, 'REVIEW', `成功: ${isbn}`);
        res.json({ success: true, message: '评价成功' });
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.get('/api/books/:isbn/review-stats', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore', 'regulator', 'user']), async (req, res) => {
    const { isbn } = req.params;
    try {
        const stats = await queryChaincode('GetReviewStats', [isbn]);
        res.json(stats);
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

app.post('/api/secondhand/list', checkRole(['user']), async (req, res) => {
    const { isbn, price, condition } = req.body;
    log(colors.yellow, 'SECONDHAND', `用户 ${req.user.name} 上架二手书: ${isbn}, 价格: ¥${price}`);
    try {
        const hasBought = purchaseRecords.some(p => p.buyer === req.user.name && p.isbn === isbn);
        if (!hasBought) return res.status(403).json({ error: '只有购买过该书的用户才能转售' });
        const result = await invokeChaincode('ListSecondHandBook', [isbn, req.user.id.toString(), req.user.name, price.toString(), condition || '九成新']);
        log(colors.green, 'SECONDHAND', `成功: ${isbn}`);
        res.json({ success: true, message: '上架成功', listingId: result });
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.get('/api/secondhand/listings', checkRole(['user', 'bookstore']), async (req, res) => {
    log(colors.cyan, 'QUERY', `用户 ${req.user.name} 查询二手书列表`);
    try {
        const listings = await queryChaincode('GetSecondHandListings', []);
        res.json(listings);
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

app.post('/api/secondhand/buy', checkRole(['user']), async (req, res) => {
    const { listingId, isbn } = req.body;
    log(colors.yellow, 'SECONDHAND', `用户 ${req.user.name} 购买二手书: ${isbn}, 挂牌ID: ${listingId}`);
    try {
        await invokeChaincode('BuySecondHandBook', [isbn, listingId, req.user.id.toString(), req.user.name]);
        log(colors.green, 'SECONDHAND', `成功: ${isbn}`);
        res.json({ success: true, message: '二手书购买成功' });
    } catch (error) {
        log(colors.red, 'ERROR', error.message);
        res.status(500).json({ error: error.message });
    }
});

app.get('/api/statistics', checkRole(['regulator']), async (req, res) => {
    log(colors.cyan, 'QUERY', `用户 ${req.user.name} 查询统计数据`);
    try {
        const stats = await queryChaincode('GetStatistics', []);
        res.json(stats);
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

app.get('/api/carbon-stats', checkRole(['regulator', 'publisher']), async (req, res) => {
    log(colors.cyan, 'QUERY', `用户 ${req.user.name} 查询碳足迹统计`);
    try {
        const books = await queryChaincode('QueryAllBooks', []);
        let totalPrint = 0, totalTransport = 0, totalEmission = 0;
        for (const book of books) {
            totalPrint += book.carbonFootprint?.printEmissions || 0;
            totalTransport += book.carbonFootprint?.transportEmissions || 0;
            totalEmission += book.carbonFootprint?.totalEmissions || 0;
        }
        res.json({ totalPrintEmissions: totalPrint, totalTransportEmissions: totalTransport, totalEmissions: totalEmission, bookCount: books.length });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

app.get('/health', (req, res) => {
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

const PORT = 3000;
app.listen(PORT, () => {
    console.log(`\n${colors.green}✅ 后端服务运行在 http://localhost:${PORT}${colors.reset}`);
    console.log(`${colors.cyan}📋 日志模式已开启，所有操作将实时显示${colors.reset}\n`);
});
