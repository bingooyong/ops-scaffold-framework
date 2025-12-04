#!/bin/bash

# 生成测试用TLS证书
# 仅用于开发环境测试，生产环境请使用正式证书

set -e

# 证书存放目录
CERT_DIR="./certs"
mkdir -p "$CERT_DIR"

echo "Generating test certificates..."

# 1. 生成CA私钥
openssl genrsa -out "$CERT_DIR/ca.key" 2048

# 2. 生成CA证书
openssl req -new -x509 -days 365 -key "$CERT_DIR/ca.key" -out "$CERT_DIR/ca.crt" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=OpsScaffold/OU=Dev/CN=OpsScaffold-CA"

# 3. 生成服务端私钥（Manager）
openssl genrsa -out "$CERT_DIR/server.key" 2048

# 4. 生成服务端证书签名请求
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=OpsScaffold/OU=Dev/CN=manager.example.com"

# 5. 使用CA签发服务端证书
openssl x509 -req -days 365 -in "$CERT_DIR/server.csr" \
    -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
    -out "$CERT_DIR/server.crt"

# 6. 生成客户端私钥（Daemon）
openssl genrsa -out "$CERT_DIR/client.key" 2048

# 7. 生成客户端证书签名请求
openssl req -new -key "$CERT_DIR/client.key" -out "$CERT_DIR/client.csr" \
    -subj "/C=CN/ST=Beijing/L=Beijing/O=OpsScaffold/OU=Dev/CN=daemon"

# 8. 使用CA签发客户端证书
openssl x509 -req -days 365 -in "$CERT_DIR/client.csr" \
    -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial \
    -out "$CERT_DIR/client.crt"

# 9. 清理临时文件
rm -f "$CERT_DIR"/*.csr "$CERT_DIR"/*.srl

echo "Test certificates generated successfully!"
echo ""
echo "Generated files:"
echo "  CA证书:      $CERT_DIR/ca.crt"
echo "  CA私钥:      $CERT_DIR/ca.key"
echo "  服务端证书:   $CERT_DIR/server.crt"
echo "  服务端私钥:   $CERT_DIR/server.key"
echo "  客户端证书:   $CERT_DIR/client.crt"
echo "  客户端私钥:   $CERT_DIR/client.key"
echo ""
echo "Update your config files to use these certificates:"
echo "  manager.tls.cert_file: $CERT_DIR/client.crt"
echo "  manager.tls.key_file:  $CERT_DIR/client.key"
echo "  manager.tls.ca_file:   $CERT_DIR/ca.crt"
