# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.148.1/containers/go/.devcontainer/base.Dockerfile

# [Choice] Go version: 1, 1.15, 1.14
ARG VARIANT="1"
FROM mcr.microsoft.com/vscode/devcontainers/go:${VARIANT}

# [Option] Install Node.js
ARG INSTALL_NODE="true"
ARG NODE_VERSION="lts/*"
RUN if [ "${INSTALL_NODE}" = "true" ]; then su vscode -c "source /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# Install Essentia for audio analysis (energy detection)
# Note: Essentia pip wheels are only available for x86_64. On ARM64, this step will be skipped.
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends python3-pip \
    && (pip3 install --break-system-packages essentia 2>/dev/null || echo "Essentia not available for this platform - energy analysis will be disabled") \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install dependencies and download custom TagLib with DSF support (same as production)
ARG CROSS_TAGLIB_VERSION=2.1.0-1
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        wget \
        pkg-config \
        build-essential \
        ffmpeg \
    && mkdir -p /usr/local/taglib \
    && cd /tmp \
    && ARCH=$(uname -m) \
    && if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi \
    && if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi \
    && TAGLIB_FILE="taglib-linux-${ARCH}.tar.gz" \
    && wget https://github.com/navidrome/cross-taglib/releases/download/v${CROSS_TAGLIB_VERSION}/${TAGLIB_FILE} \
    && tar -xzf ${TAGLIB_FILE} -C /usr/local/taglib \
    && chmod -R 755 /usr/local/taglib \
    && sed -i 's|^prefix=/taglib|prefix=/usr/local/taglib|g' /usr/local/taglib/lib/pkgconfig/taglib.pc \
    && sed -i 's|^prefix=/taglib|prefix=/usr/local/taglib|g' /usr/local/taglib/lib/pkgconfig/taglib_c.pc \
    && echo "export PKG_CONFIG_PATH=/usr/local/taglib/lib/pkgconfig:$PKG_CONFIG_PATH" >> /etc/environment \
    && echo "export LD_LIBRARY_PATH=/usr/local/taglib/lib:$LD_LIBRARY_PATH" >> /etc/environment \
    && rm -f ${TAGLIB_FILE} \
    && apt-get -y autoremove \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set PKG_CONFIG_PATH for the current session and future sessions
ENV PKG_CONFIG_PATH=/usr/local/taglib/lib/pkgconfig:$PKG_CONFIG_PATH
ENV LD_LIBRARY_PATH=/usr/local/taglib/lib:$LD_LIBRARY_PATH

# [Optional] Uncomment the next line to use go get to install anything else you need
# RUN go get -x <your-dependency-or-tool>

# [Optional] Uncomment this line to install global node packages.
# RUN su vscode -c "source /usr/local/share/nvm/nvm.sh && npm install -g <your-package-here>" 2>&1

RUN echo "Devcontainer is ready. Run `make dev` to start (hot-reloading) servers."
