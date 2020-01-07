FROM ubuntu:18.04

RUN set -ex; \
    groupadd -r buildkite --gid=9999; \
    useradd -r -g buildkite --uid=9999 buildkite; \
    mkdir -p /home/buildkite; \
    chown -R buildkite: /home/buildkite

ENV HOME /home/buildkite

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
      curl \
      ca-certificates \
      bash \
      git \
      perl \
      rsync \
      openssh-client \
      curl \
      docker.io \
      jq \
    && rm -rf /var/lib/apt/lists/*

RUN curl -Lfs -o /sbin/tini  https://github.com/krallin/tini/releases/download/v0.18.0/tini \
    && chmod +x /sbin/tini \
    && curl -Lfs https://github.com/docker/compose/releases/download/1.24.0/docker-compose-Linux-x86_64 -o /usr/local/bin/docker-compose \
    && chmod +x /usr/local/bin/docker-compose

ENV BUILDKITE_AGENT_CONFIG=/buildkite/buildkite-agent.cfg \
    PATH="/usr/local/bin:${PATH}"

RUN mkdir -p /buildkite/builds /buildkite/hooks /buildkite/plugins \
    && curl -Lfs -o /usr/local/bin/ssh-env-config.sh https://raw.githubusercontent.com/buildkite/docker-ssh-env-config/master/ssh-env-config.sh \
    && chmod +x /usr/local/bin/ssh-env-config.sh

COPY ./buildkite-agent.cfg /buildkite/buildkite-agent.cfg
COPY ./buildkite-agent /usr/local/bin/buildkite-agent
COPY ./entrypoint.sh /usr/local/bin/buildkite-agent-entrypoint

# Install kubectl + jsonnet + bazel
RUN set -ex; \
    apt-get update; \
    apt-get install -y \
      apt-transport-https \
      gnupg \
      ; \
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -; \
    echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list; \
    curl https://bazel.build/bazel-release.pub.gpg | apt-key add -; \
    echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list; \
    apt-get update; \
    apt-get install -y \
      bazel \
      kubectl \
      ; \
    rm -rf /var/lib/apt/lists/*; \
    curl -LkSs -o/tmp/jsonnet.tar.gz https://github.com/google/jsonnet/releases/download/v0.14.0/jsonnet-bin-v0.14.0-linux.tar.gz; \
    tar zxvf /tmp/jsonnet.tar.gz; \
    chmod +x jsonnet; \
    mv jsonnet /usr/local/bin/jsonnet

VOLUME /buildkite
ENTRYPOINT ["buildkite-agent-entrypoint"]
CMD ["start"]
