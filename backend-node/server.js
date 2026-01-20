import grpc from '@grpc/grpc-js';
import protoLoader from '@grpc/proto-loader';
import { OAuth2Client } from 'google-auth-library';
import pg from 'pg';
import dotenv from 'dotenv';
import { fileURLToPath } from 'url';
import path from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

dotenv.config();

const { Pool } = pg;

// Database setup
const pool = new Pool({
    connectionString: process.env.DATABASE_URL || 'postgresql://doric:doric@db:5432/doric'
});

// Initialize DB schema
await pool.query(`
  CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT,
    picture TEXT
  );
`);

// Google OAuth setup
const oauth2Client = new OAuth2Client(
    process.env.GOOGLE_CLIENT_ID,
    process.env.GOOGLE_CLIENT_SECRET,
    'http://localhost:4242/callback'
);

// Load proto
const PROTO_PATH = path.resolve(__dirname, './proto/doric.proto');
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true
});
const doricProto = grpc.loadPackageDefinition(packageDefinition).doric;

// gRPC service implementation
const authService = {
    GetAuthUrl: (call, callback) => {
        const url = oauth2Client.generateAuthUrl({
            access_type: 'offline',
            scope: [
                'https://www.googleapis.com/auth/userinfo.profile',
                'https://www.googleapis.com/auth/userinfo.email'
            ]
        });
        callback(null, { url });
    },

    ExchangeCode: async (call, callback) => {
        try {
            const { code } = call.request;
            const { tokens } = await oauth2Client.getToken(code);
            oauth2Client.setCredentials(tokens);

            const res = await oauth2Client.request({
                url: 'https://www.googleapis.com/oauth2/v3/userinfo'
            });

            const userInfo = res.data;

            // Upsert user to database
            await pool.query(
                `INSERT INTO users (id, email, name, picture) 
         VALUES ($1, $2, $3, $4) 
         ON CONFLICT (id) DO UPDATE 
         SET email = $2, name = $3, picture = $4`,
                [userInfo.sub, userInfo.email, userInfo.name, userInfo.picture]
            );

            callback(null, {
                success: true,
                user: {
                    id: userInfo.sub,
                    email: userInfo.email,
                    name: userInfo.name,
                    picture: userInfo.picture
                }
            });
        } catch (error) {
            console.error('Auth error:', error);
            callback(null, {
                success: false,
                error: error.message
            });
        }
    },

    GetCurrentUser: async (call, callback) => {
        callback(null, { id: '', email: '', name: '', picture: '' });
    }
};

// Start gRPC server
const server = new grpc.Server();
server.addService(doricProto.AuthService.service, authService);
server.bindAsync(
    '0.0.0.0:50051',
    grpc.ServerCredentials.createInsecure(),
    (err, port) => {
        if (err) {
            console.error('Failed to start server:', err);
            return;
        }
        console.log(`✅ Doric Backend (Node.js) listening on port ${port}`);
        server.start();
    }
);
