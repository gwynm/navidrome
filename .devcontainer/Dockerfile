# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.148.1/containers/go/.devcontainer/base.Dockerfile

# [Choice] Go version: 1, 1.15, 1.14
ARG VARIANT="1"
FROM mcr.microsoft.com/vscode/devcontainers/go:${VARIANT}

# [Option] Install Node.js
ARG INSTALL_NODE="true"
ARG NODE_VERSION="lts/*"
RUN if [ "${INSTALL_NODE}" = "true" ]; then su vscode -c "source /usr/local/share/nvm/nvm.sh && nvm install ${NODE_VERSION} 2>&1"; fi

# Install system dependencies for Homebrew and Python fallback
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        curl \
        git \
        procps \
        file \
        python3-pip \
        python3-dev \
        build-essential \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Homebrew as vscode user
USER vscode
WORKDIR /home/vscode

# Install Homebrew
RUN NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Add Homebrew to PATH for this build and set up shell
RUN echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> ~/.bashrc

# Install Essentia via Homebrew (builds from source, native ARM64 support)
# This may take a while as it compiles from source
RUN eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)" \
    && brew tap MTG/essentia \
    && brew install essentia --HEAD \
    || echo "Essentia Homebrew build failed - will use Python fallback if available"

# Set up Homebrew environment variables
ENV PATH="/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin:${PATH}"
ENV HOMEBREW_PREFIX="/home/linuxbrew/.linuxbrew"
ENV HOMEBREW_CELLAR="/home/linuxbrew/.linuxbrew/Cellar"
ENV HOMEBREW_REPOSITORY="/home/linuxbrew/.linuxbrew/Homebrew"

# Switch back to root for remaining setup
USER root

# Install dependencies and download custom TagLib with DSF support (same as production)
ARG CROSS_TAGLIB_VERSION=2.1.0-1
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        wget \
        pkg-config \
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

RUN echo "Devcontainer is ready. Run 'make dev' to start (hot-reloading) servers."
