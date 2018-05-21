FROM ubuntu:16.04
RUN apt-get update && apt-get install vim net-tools iputils-ping wget curl git build-essential tcpdump unzip -y
RUN wget https://redirector.gvt1.com/edgedl/go/go1.9.2.linux-amd64.tar.gz 
RUN tar -C /usr/local -xzf go*.tar.gz
RUN echo "export PATH=$PATH:/usr/local/go/bin:/root/go/bin" >> /root/.bashrc
RUN echo "export TERM=xterm-256color" >> /root/.bashrc
RUN git clone https://github.com/connorwstein/dotfiles 
RUN cd dotfiles && ./setup
RUN /usr/local/go/bin/go get -u github.com/golang/glog
RUN /usr/local/go/bin/go get -u github.com/derekparker/delve/cmd/dlv
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"
CMD tail -f /dev/null
