FROM centos:7.7.1908

RUN set -ex; \
    groupadd -r buildkite --gid=9999; \
	useradd -r -g buildkite --uid=9999 buildkite; \
	mkdir -p /home/buildkite; \
	chown -R buildkite: /home/buildkite

ENV HOME /home/buildkite

RUN yum -y install \
      curl \
      ca-certificates \
      bash \
      git \
      perl \
      rsync \
      openssh-clients \
      curl \
      crontabs \
      epel-release \
    && yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo \
    && yum -y install docker-ce jq \
    && yum clean all \
    && rm -rf /var/cache/yum*

RUN curl -Lfs -o /sbin/tini  https://github.com/krallin/tini/releases/download/v0.18.0/tini \
    && chmod +x /sbin/tini \
    && curl -Lfs https://github.com/docker/compose/releases/download/1.24.0/docker-compose-Linux-x86_64 -o /usr/local/bin/docker-compose \
    && chmod +x /usr/local/bin/docker-compose

ENV BUILDKITE_AGENT_CONFIG=/buildkite/buildkite-agent.cfg \
    PATH="/usr/local/bin:${PATH}"

RUN mkdir -p /buildkite/builds /buildkite/hooks /buildkite/plugins \
    && curl -Lfs -o /usr/local/bin/ssh-env-config.sh https://raw.githubusercontent.com/buildkite/docker-ssh-env-config/master/ssh-env-config.sh \
    && chmod +x /usr/local/bin/ssh-env-config.sh

RUN set -ex; \
    curl -kSs -o/etc/yum.repos.d/bazel.repo https://copr.fedorainfracloud.org/coprs/vbatts/bazel/repo/epel-7/vbatts-bazel-epel-7.repo; \
    yum install -y \
	  bazel \
	  gcc \
	  ; \
    yum clean all; \
    rm -rf /var/cache/yum*

COPY ./buildkite-agent.cfg /buildkite/buildkite-agent.cfg
COPY ./buildkite-agent /usr/local/bin/buildkite-agent
COPY ./entrypoint.sh /usr/local/bin/buildkite-agent-entrypoint

VOLUME /buildkite
ENTRYPOINT ["buildkite-agent-entrypoint"]
CMD ["start"]
