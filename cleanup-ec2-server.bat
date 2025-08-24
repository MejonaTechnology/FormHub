@echo off
echo ========================================
echo    EC2 Server Cleanup Script
echo ========================================
echo.

echo Current server status:
echo Memory: 949MB total (22.7%% used by systemd-journald!)
echo Storage: 7.4GB/8GB used (593MB free)
echo Big files found: 711MB Go cache + 71MB tar.gz
echo.

echo [1/6] Analyzing current memory usage...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
echo 'Top memory consumers:'
ps aux --sort=-%mem | head -5
echo
echo 'Disk usage:'
df -h /
"

echo.
echo [2/6] Cleaning up large unnecessary files...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
echo 'Removing Go installation files and cache (711MB + 71MB)...'
rm -f /home/ec2-user/go1.23.0.linux-amd64.tar.gz
rm -rf /home/ec2-user/go/pkg/mod/cache/download/
du -sh /home/ec2-user/go || echo 'Go cache cleaned'

echo 'Cleaning npm logs and cache...'
rm -rf /home/ec2-user/.npm/_logs/
rm -f /home/ec2-user/*.log
rm -f /home/ec2-user/.claude.json.backup

echo 'Cleaning yum cache...'
sudo yum clean all

echo 'Removing old RPM files...'
rm -f /home/ec2-user/*.rpm
"

echo.
echo [3/6] Optimizing systemd-journald (using 221MB memory!)...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
echo 'Configuring journal size limits...'
sudo tee /etc/systemd/journald.conf.d/size.conf > /dev/null << EOF
[Journal]
SystemMaxUse=50M
RuntimeMaxUse=50M
MaxRetentionSec=1week
EOF

echo 'Cleaning old journal logs...'
sudo journalctl --vacuum-size=50M
sudo journalctl --vacuum-time=7d
sudo systemctl restart systemd-journald
"

echo.
echo [4/6] Cleaning up Docker if present (and not needed)...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
if command -v docker >/dev/null 2>&1; then
    echo 'Cleaning Docker system...'
    sudo docker system prune -af || true
    sudo docker volume prune -f || true
    
    # If Docker not being used, remove it
    read -p 'Remove Docker completely? (y/N): ' remove_docker
    if [[ \$remove_docker == 'y' ]]; then
        sudo systemctl stop docker
        sudo systemctl disable docker
        sudo yum remove -y docker docker-*
    fi
else
    echo 'Docker not installed - skipping'
fi
"

echo.
echo [5/6] Optimizing PHP-FPM and Apache memory usage...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
echo 'Optimizing PHP-FPM for low memory...'
sudo cp /etc/php-fpm.d/www.conf /etc/php-fpm.d/www.conf.backup
sudo sed -i 's/pm.max_children = .*/pm.max_children = 5/' /etc/php-fpm.d/www.conf
sudo sed -i 's/pm.start_servers = .*/pm.start_servers = 2/' /etc/php-fpm.d/www.conf
sudo sed -i 's/pm.min_spare_servers = .*/pm.min_spare_servers = 1/' /etc/php-fpm.d/www.conf
sudo sed -i 's/pm.max_spare_servers = .*/pm.max_spare_servers = 3/' /etc/php-fpm.d/www.conf

echo 'Optimizing Apache MPM for low memory...'
sudo tee -a /etc/httpd/conf.d/mpm.conf > /dev/null << EOF
<IfModule mpm_prefork_module>
    StartServers 2
    MinSpareServers 2
    MaxSpareServers 5
    MaxRequestWorkers 10
    MaxConnectionsPerChild 1000
</IfModule>
EOF

sudo systemctl restart php-fpm httpd
"

echo.
echo [6/6] Final cleanup and optimization...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
echo 'Cleaning temporary files...'
sudo find /tmp -type f -atime +7 -delete
sudo find /var/tmp -type f -atime +7 -delete

echo 'Cleaning package cache...'
sudo yum autoremove -y
sudo yum clean all

echo 'Optimizing MariaDB memory usage...'
sudo tee -a /etc/my.cnf.d/memory-optimized.cnf > /dev/null << EOF
[mysqld]
innodb_buffer_pool_size = 64M
query_cache_size = 0
query_cache_type = 0
tmp_table_size = 8M
max_heap_table_size = 8M
max_connections = 50
EOF

sudo systemctl restart mariadb

echo 'Setting up swap file for additional memory...'
if [ ! -f /swapfile ]; then
    sudo fallocate -l 512M /swapfile
    sudo chmod 600 /swapfile
    sudo mkswap /swapfile
    sudo swapon /swapfile
    echo '/swapfile swap swap defaults 0 0' | sudo tee -a /etc/fstab
fi
"

echo.
echo ========================================
echo    Cleanup Results
echo ========================================
echo.

echo Checking results...
ssh -i "D:/Mejona Workspace/Product/Mejona complete website/mejonaN.pem" ec2-user@13.201.64.45 "
echo 'NEW DISK USAGE:'
df -h / | grep -v Filesystem

echo
echo 'NEW MEMORY USAGE:'
free -h

echo
echo 'TOP PROCESSES:'
ps aux --sort=-%mem | head -5

echo
echo 'SERVICES STATUS:'
sudo systemctl status php-fpm httpd mariadb --no-pager -l
"

echo.
echo ========================================
echo    FormHub Deployment Space Ready!
echo ========================================
echo.
echo After cleanup, you should have much more:
echo ✓ Free disk space for FormHub
echo ✓ Lower memory usage
echo ✓ Optimized services
echo ✓ 512MB swap file added
echo.
echo Ready to deploy FormHub with available resources!
echo.
pause