# 查看 系统的资源限制
echo "=== 系统环境信息 ==="
echo "CPU 核心数: $(nproc)"
echo "总内存: $(free -h | grep Mem | awk '{print $2}')"
echo "磁盘空间: $(df -h / | tail -1 | awk '{print $4}') 可用"

# 查看当前负载
echo "当前负载: $(uptime | awk -F'load average:' '{print $2}')"


# find . -type f -name ".*.metadata" -delete

# find / -name ".*.metadata" -type f 2>/dev/null | xargs rm
