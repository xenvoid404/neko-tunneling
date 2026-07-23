# ~/.profile: executed by Bourne-compatible login shells.

if [ "$BASH" ]; then
  if [ -f ~/.bashrc ]; then
    . ~/.bashrc
  fi
fi

mesg n || true
ln -fs /usr/share/zoneinfo/Asia/Jakarta /etc/localtime
clear
neofetch