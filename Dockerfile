FROM --platform=linux/amd64 golang:1.24-bookworm

# 设置时区为上海时区
RUN apt-get update && apt-get install -y tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    dpkg-reconfigure -f noninteractive tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# 保持容器常驻
CMD ["sleep", "infinity"]


