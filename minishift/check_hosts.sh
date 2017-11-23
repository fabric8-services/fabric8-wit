IP=$(minishift ip)
HOSTNAME=minishift.local
HOSTNAME_REGEX='minishift\.local'
ETC_HOSTS=/etc/hosts
if grep -q "$IP $HOSTNAME" $ETC_HOSTS; then
    echo "Hosts entry exists"
else
    echo "Updating /etc/hosts (Remove old minishift.local if any and add new one)"
    # remove old entries with $HOSTNAME if any
    sudo sed -i "/$HOSTNAME_REGEX/d" $ETC_HOSTS
    # add new entry
    echo $IP $HOSTNAME | sudo tee --append $ETC_HOSTS;
fi
