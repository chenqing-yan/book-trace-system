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

// 用户数据库
const users = {
    'publisher': { id: 1, name: '机械工业出版社', role: 'publisher', roleName: '出版社', password: '123' },
    'printer': { id: 2, name: '新华印刷厂', role: 'printer', roleName: '印刷厂', password: '123' },
    'wholesaler': { id: 3, name: '新华批发商', role: 'wholesaler', roleName: '批发商', password: '123' },
    'bookstore': { id: 4, name: '新华书店', role: 'bookstore', roleName: '书店', password: '123' },
    'regulator': { id: 5, name: '新闻出版署', role: 'regulator', roleName: '监管机构', password: '123' },
    'user': { id: 6, name: '普通用户', role: 'user', roleName: '普通用户', password: '123' }
};

// 购买记录存储
let purchaseRecords = [];

// TLS 环境配置
const fabricEnv = {
    PATH: process.env.PATH,
    FABRIC_CFG_PATH: '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/config',
    CORE_PEER_TLS_ENABLED: 'true',
    CORE_PEER_LOCALMSPID: 'Org1MSP',
    CORE_PEER_TLS_ROOTCERT_FILE: '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt',
    CORE_PEER_MSPCONFIGPATH: '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp',
    CORE_PEER_ADDRESS: 'localhost:7051'
};

const ordererCA = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem';
const peer0CA = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt';
const peer1CA = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt';
const baseDir = '/home/hongyingying3232705141/HyperledgerFabric/fabric-samples/test-network';

// 辅助函数：查询链码
async function queryChaincode(func, args) {
    const command = `cd ${baseDir} && peer chaincode query -C book-trace-channel -n booktrace -c '{"Args":["${func}","${args.join('","')}"]}' --tls --cafile ${ordererCA}`;
    const { stdout } = await execPromise(command, { env: fabricEnv });
    let result = stdout.trim();
    if (result.includes('Query result: ')) {
        result = result.split('Query result: ')[1];
    }
    return JSON.parse(result);
}

// 辅助函数：调用链码
async function invokeChaincode(func, args) {
    const command = `cd ${baseDir} && peer chaincode invoke -C book-trace-channel -n booktrace -c '{"Args":["${func}","${args.join('","')}"]}' --tls --cafile ${ordererCA} --peerAddresses localhost:7051 --tlsRootCertFiles ${peer0CA} --peerAddresses localhost:9051 --tlsRootCertFiles ${peer1CA} --waitForEvent`;
    const { stdout } = await execPromise(command, { env: fabricEnv });
    return stdout;
}

// 权限检查中间件
function checkRole(allowedRoles) {
    return (req, res, next) => {
        const token = req.headers.authorization?.split(' ')[1];
        if (!token) {
            return res.status(401).json({ error: '未登录' });
        }
        try {
            const decoded = jwt.verify(token, SECRET_KEY);
            if (!allowedRoles.includes(decoded.role)) {
                return res.status(403).json({ error: '权限不足' });
            }
            req.user = decoded;
            next();
        } catch (error) {
            res.status(401).json({ error: '登录已过期' });
        }
    };
}

// ==================== API 路由 ====================

// 登录
app.post('/api/login', (req, res) => {
    const { username, password } = req.body;
    const user = users[username];
    if (user && user.password === password) {
        const token = jwt.sign({ id: user.id, name: user.name, role: user.role, roleName: user.roleName }, SECRET_KEY, { expiresIn: '8h' });
        res.json({ success: true, token, user: { name: user.name, role: user.role, roleName: user.roleName } });
    } else {
        res.status(401).json({ error: '用户名或密码错误' });
    }
});

// 获取当前用户信息
app.get('/api/me', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore', 'regulator', 'user']), (req, res) => {
    res.json(req.user);
});

// 查询图书列表（根据角色过滤）
app.get('/api/books', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore', 'regulator', 'user']), async (req, res) => {
    try {
        const books = await queryChaincode('QueryAllBooks', []);
        
        // 根据角色过滤图书
        let filteredBooks = books;
        if (req.user.role === 'printer') {
            filteredBooks = books.filter(b => b.currentState === '图书信息已录入' || b.currentState === '印刷完成');
        } else if (req.user.role === 'wholesaler') {
            filteredBooks = books.filter(b => b.currentState === '出厂分发' || b.currentState === '批发商入库');
        } else if (req.user.role === 'bookstore') {
            // 书店：显示待上架、在售中、已售出的图书
            filteredBooks = books.filter(b => 
                b.currentState === '配送到门店' || 
                b.currentState === '书店上架' || 
                b.currentState === '已售出'
            );
        } else if (req.user.role === 'user') {
            filteredBooks = books.filter(b => b.currentState === '书店上架');
        }
        // publisher 和 regulator 看到所有图书
        
        res.json(filteredBooks);
    } catch (error) {
        console.error('查询图书失败:', error);
        res.status(500).json({ error: error.message });
    }
});

// 查询单本图书
app.get('/api/books/:isbn', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore', 'regulator', 'user']), async (req, res) => {
    try {
        const { isbn } = req.params;
        const book = await queryChaincode('QueryBookByISBN', [isbn]);
        res.json(book);
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

// 创建图书（仅出版社）
app.post('/api/books', checkRole(['publisher']), async (req, res) => {
    try {
        const { isbn, bookName, author, publisher, publishDate, category, price } = req.body;
        await invokeChaincode('CreateBook', [isbn, bookName, author, publisher, publishDate, category, price]);
        res.json({ success: true, message: '图书创建成功' });
    } catch (error) {
        console.error('创建图书失败:', error);
        res.status(500).json({ error: error.message });
    }
});

// 更新图书状态
app.put('/api/books/:isbn/state', checkRole(['publisher', 'printer', 'wholesaler', 'bookstore']), async (req, res) => {
    try {
        const { isbn } = req.params;
        const { newState, remark } = req.body;
        const orgName = req.user.roleName;
        await invokeChaincode('UpdateBookState', [isbn, newState, orgName, remark]);
        res.json({ success: true, message: '状态更新成功' });
    } catch (error) {
        console.error('更新状态失败:', error);
        res.status(500).json({ error: error.message });
    }
});

// 购买图书（普通用户）
app.post('/api/books/:isbn/buy', checkRole(['user']), async (req, res) => {
    try {
        const { isbn } = req.params;
        const book = await queryChaincode('QueryBookByISBN', [isbn]);
        
        if (book.currentState !== '书店上架') {
            return res.status(400).json({ error: '该书当前不可购买' });
        }
        
        await invokeChaincode('UpdateBookState', [isbn, '已售出', req.user.roleName, `用户 ${req.user.name} 购买`]);
        
        purchaseRecords.push({
            id: Date.now(),
            isbn: book.isbn,
            bookName: book.bookName,
            buyer: req.user.name,
            buyTime: new Date().toISOString(),
            price: book.price
        });
        
        res.json({ success: true, message: '购买成功！' });
    } catch (error) {
        console.error('购买失败:', error);
        res.status(500).json({ error: error.message });
    }
});

// 获取我的购买记录（普通用户）
app.get('/api/my-purchases', checkRole(['user']), async (req, res) => {
    const myPurchases = purchaseRecords.filter(p => p.buyer === req.user.name);
    res.json(myPurchases);
});

// 获取统计数据（仅监管机构）
app.get('/api/statistics', checkRole(['regulator']), async (req, res) => {
    try {
        const stats = await queryChaincode('GetStatistics', []);
        res.json(stats);
    } catch (error) {
        console.error('获取统计失败:', error);
        res.status(500).json({ error: error.message });
    }
});

// 健康检查
app.get('/health', (req, res) => {
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

const PORT = 3000;
app.listen(PORT, () => {
    console.log(`✅ 后端服务运行在 http://localhost:${PORT}`);
    console.log('   测试账号: publisher/123, printer/123, wholesaler/123, bookstore/123, regulator/123, user/123');
});
