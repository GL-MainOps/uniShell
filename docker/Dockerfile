FROM golang:alpine

RUN apk add --no-cache zsh fish nushell bash bash-completion git tmux zellij  vim openssh curl wget

ENV SHELL=/bin/bash
ENV EDITOR=vim
ENV TERM=xterm-256color
ENV PATH="/tmp/uHome/bin:/tmp/app/bin:${PATH}"

# RUN printf "export PATH=\"/tmp/uHome/bin:/tmp/app/bin:\$PATH\"\nalias rr='rm -rfv tmp/* && sleep 1 && clear'\nalias cc='clear'\nalias tmux='tmux start-server \\\\; set-option -g default-command \"bash --rcfile /tmp/uHome/.bashrc\" \\\\; new-session -d -s mysession \\\\; attach-session -t mysession'\nPS1='\$ '\nexport PS1\n" > /tmp/uHome/.bashrc  && chmod 777 /tmp/uHome/.bashrc

# RUN echo -e "set mouse-=a\nset number" > /.vimrc \
#     && echo -e "set mouse-=a\nset number" > /root/.vimrc \
#     && chmod 777 /.vimrc /root/.vimrc

#COPY ./vimrc.local /root/.vimrc
#COPY ./vimrc.local /tmp/.vimrc

# RUN chmod 777 /tmp/.vimrc /root/.vimrc

ARG UID=1000
RUN adduser -D -u ${UID} -G users -h /tmp/uHome -s /bin/bash appuser
USER appuser
