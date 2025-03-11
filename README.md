# Rhaein: Rust-Based Discord Radio Streaming Bot Project Plan

## Project Summary

Rhaein is a lightweight Discord bot designed for 24/7 radio streaming from listen.moe with embedded dependencies and ARM-optimized deployment. The bot will feature:

-   **Zero-dependency Opus streaming** through direct DCA passthrough
-   **Automatic reconnection** with jitter compensation
-   **Hardware-optimized builds** for Raspberry Pi 3/4/5
-   **Dynamic bitrate adjustment** based on network conditions
-   **Status monitoring** through Prometheus metrics

## Technical Stack

### Core Dependencies

| Component       | Crate           | Purpose              | Embedded |
| --------------- | --------------- | -------------------- | -------- |
| Discord Gateway | Serenity 0.13.0 | Bot framework        | Yes      |
| Voice Engine    | Songbird 0.7.2  | Audio pipeline       | Yes      |
| Audio Decoding  | Symphonia 0.7.0 | Opus/MP3 decoding    | Yes      |
| Streaming       | Reqwest 0.12.2  | HTTP audio streaming | Yes      |
| Configuration   | Figment 0.6.1   | TOML/YAML configs    | Yes      |

### Build Toolchain

1. **Cross-compilation**: `cargo-zigbuild` for ARMv7/ARM64 static binaries
2. **Dependency Bundling**: Static linking with `audiopus-sys` + `openssl-sys`
3. **Size Optimization**: `strip = true` + `lto = "thin"` in Cargo.toml
4. **Pi Deployment**: `systemd` service files with watchdog integration

## Folder Structure

```md
rhaein/
├── Cargo.toml
├── .cargo/
│ └── config.toml # Cross-compile profiles
├── src/
│ ├── main.rs # Bot initialization
│ ├── commands/ # (Mod) Discord slash commands
│ │ └── voice.rs # /play, /stop, /status
│ ├── audio/ # (Mod) Streaming logic
│ │ ├── streamer.rs # Audio buffer management
│ │ └── dca_passthrough.rs # Opus packet handler
│ └── config/ # (Mod) Configuration loader
├── resources/
│ ├── rhaein.service # systemd unit file
│ └── default.toml # Base configuration
└── scripts/
└── deploy_pi.sh # ARM deployment script
```

## Development Roadmap

### Phase 1: Core Voice Integration (Foundation)

-   [ ] Serenity/Songbird project initialization
-   [ ] Voice channel join/leave handling
-   [ ] Basic audio stream playback skeleton

### Phase 2: Listen.moe Streaming Implementation

-   [ ] Opus-over-HTTP direct passthrough
-   [ ] Stream metadata extraction (current track)
-   [ ] Adaptive jitter buffer implementation

### Phase 3: ARM Deployment Packaging

-   [ ] MUSL static builds with cargo-zigbuild
-   [ ] Systemd service file integration
-   [ ] Resource monitoring dashboard setup

### Phase 4: User Interaction Layer

-   [ ] Slash command registration
-   [ ] Dynamic bitrate adjustment commands
-   [ ] Network health status reporting

### Phase 5: Productionization

-   [ ] CI/CD pipeline for ARM builds
-   [ ] Automated recovery mechanisms
-   [ ] Documentation & deployment guides

## Dependency Management Strategy

1. **Static Linking**: All audio codecs compiled into final binary via:

    ```toml
    [dependencies.audiopus-sys]
    features = ["static"]
    ```

2. **Cross-Compile Cache**: Shared cargo registry using:

    ```bash
    export CARGO_TARGET_DIR="$HOME/.cargo/rhaein_target"
    ```

3. **Size Reduction**: Post-build processing with:

    ```bash
    cargo build --release && strip -s target/release/rhaein
    ```
