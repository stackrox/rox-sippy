FROM registry.access.redhat.com/ubi9/ubi:latest AS builder
WORKDIR /go/src/sippy
COPY . .
ENV PATH="/go/bin:${PATH}"
ENV GOPATH="/go"

# Install build dependencies and build both frontend and backend
RUN dnf module enable nodejs:20 -y && \
    dnf install -y go make npm && \
    dnf clean all && \
    rm -rf /var/cache/dnf

# Build frontend first (required for go:embed)
RUN cd sippy-ng && npm ci --no-audit --ignore-scripts && npm run build

# Build ACS Sippy binary
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
ARG GIT_TREE_STATE=unknown
RUN go build \
    -ldflags "-X github.com/openshift/sippy/pkg/version.commitFromGit=${GIT_COMMIT} -X github.com/openshift/sippy/pkg/version.buildDate=${BUILD_DATE} -X github.com/openshift/sippy/pkg/version.gitTreeState=${GIT_TREE_STATE}" \
    -mod=vendor \
    -o ./acs-sippy \
    ./cmd/sippy

FROM registry.access.redhat.com/ubi9/ubi:latest AS runtime
RUN mkdir -p /config

# Copy the ACS Sippy binary
COPY --from=builder /go/src/sippy/acs-sippy /bin/acs-sippy

# Copy ACS configuration files
COPY --from=builder /go/src/sippy/config/acs.yaml /config/
COPY --from=builder /go/src/sippy/config/acs-views.yaml /config/

EXPOSE 8080
ENTRYPOINT ["/bin/acs-sippy"]
CMD ["serve"]
