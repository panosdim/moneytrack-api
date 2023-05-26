#!/bin/bash
if [ "$(id -u)" != "0" ]; then
    exec sudo bash "$0" "$@"
fi

is_service_exists() {
    x="$1"
    if systemctl status "${x}" 2>/dev/null | grep -Fq "Active:"; then
        return 0
    else
        return 1
    fi
    unset x
}

INSTALL_PATH=/opt/moneytrack
EXEC_NAME=moneytrack
SERVICE_NAME=$EXEC_NAME.service
GO_CMD=/usr/local/go/bin/go
NGINX_CONF_PATH=/etc/nginx/conf.d

# Build software

$GO_CMD build -o $EXEC_NAME
ret_code=$?
if [ $ret_code != 0 ]; then
    printf "Error: [%d] when building executable. Check that you have go tools installed." $ret_code
    exit $ret_code
fi

# Check if needed files exist
if [ -f .env ] && [ -f $EXEC_NAME ] && [ -f $SERVICE_NAME ]; then
    # Check if we upgrade or install for first time
    if is_service_exists "$SERVICE_NAME"; then
        systemctl stop $SERVICE_NAME
        cp $EXEC_NAME $INSTALL_PATH
        cp .env $INSTALL_PATH
        systemctl start $SERVICE_NAME
    else
        mkdir -p $INSTALL_PATH
        cp $EXEC_NAME $INSTALL_PATH
        cp .env $INSTALL_PATH
        cp $SERVICE_NAME /usr/lib/systemd/system
        systemctl start $SERVICE_NAME
        systemctl enable $SERVICE_NAME
        sudo cp moneytrack.conf $NGINX_CONF_PATH
        sudo nginx -s reload
    fi
else
    echo "Not all needed files found. Installation failed."
    exit 1
fi
