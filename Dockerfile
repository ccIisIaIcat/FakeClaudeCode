FROM --platform=linux/amd64 golang:1.24-bookworm

# locale 与时区可按需开启
# RUN apt-get update && apt-get install -y tzdata && \
#     ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
#     dpkg-reconfigure -f noninteractive tzdata && \
#     rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# 保持容器常驻
CMD ["sleep", "infinity"]


