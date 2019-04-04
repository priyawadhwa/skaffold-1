# Copyright 2019 The Skaffold Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM gcr.io/gcp-runtimes/ubuntu_16_0_4 as runtime_deps

RUN apt-get update && \
  apt-get install --no-install-recommends --no-install-suggests -y \
  python && \
  rm -rf /var/lib/apt/lists/*

ENV KUBECTL_VERSION v1.12.0
RUN curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl  && \
  chmod +x /usr/local/bin/kubectl

ENV CLOUD_SDK_VERSION 217.0.0
RUN curl -LO https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${CLOUD_SDK_VERSION}-linux-x86_64.tar.gz && \
  tar -zxvf google-cloud-sdk-${CLOUD_SDK_VERSION}-linux-x86_64.tar.gz && \
  CLOUDSDK_PYTHON="python2.7" /google-cloud-sdk/install.sh --usage-reporting=false \
  --bash-completion=false \
  --disable-installation-options && \
  rm -rf google-cloud-sdk-*.tar.gz
ENV PATH=$PATH:/google-cloud-sdk/bin
RUN /google-cloud-sdk/bin/gcloud auth configure-docker


ENV PATH /usr/local/go/bin:/go/bin:/google-cloud-sdk/bin:$PATH

FROM runtime_deps as builder

RUN apt-get update && apt-get install --no-install-recommends --no-install-suggests -y \
  ca-certificates \
  curl \
  build-essential \
  git \
  gcc \
  python-setuptools \
  lsb-release \
  openjdk-8-jdk \
  software-properties-common \
  apt-transport-https && \
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add - && \
  apt-key fingerprint 0EBFCD88 && \
  add-apt-repository \
  "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
  xenial \
  edge" && \
  apt-get -y update && \
  apt-get -y install docker-ce=17.12.0~ce-0~ubuntu && \
  rm -rf /var/lib/apt/lists/*

COPY --from=golang:1.11 /usr/local/go /usr/local/go
ENV PATH /usr/local/go/bin:/go/bin:$PATH
ENV GOPATH /go/

WORKDIR /go/src/github.com/GoogleContainerTools/skaffold

COPY . .

FROM builder as integration
ARG VERSION

RUN make out/skaffold-linux-amd64 VERSION=$VERSION && mv out/skaffold-linux-amd64 /usr/bin/skaffold

CMD ["make", "integration"]

FROM runtime_deps as distribution

COPY --from=integration /usr/bin/skaffold /usr/bin/skaffold
