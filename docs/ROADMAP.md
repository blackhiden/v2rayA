# Roadmap & Future Work

## Overview

This document outlines the current development roadmap, known issues, technical debt, and planned features for v2rayA. Priorities may shift based on community feedback and security considerations.

## Current Version (v2rayA 2.x)

### Completed Features

- ✅ Multi-protocol support (VMess, VLESS, SS, SSR, Trojan, TUIC, Juicity)
- ✅ Web-based GUI with Vue.js
- ✅ Transparent proxy support (TProxy)
- ✅ System proxy configuration (Windows/macOS)
- ✅ Subscription management with auto-update
- ✅ Routing rules (RoutingA)
- ✅ DNS configuration and anti-pollution
- ✅ Multi-language support
- ✅ Cross-platform compatibility
- ✅ Docker support
- ✅ JWT authentication

## Short-term Roadmap (Next 3-6 months)

### High Priority

1. **Security Enhancements**
   - [ ] Implement rate limiting for API endpoints
   - [ ] Add CSRF protection
   - [ ] Enhance input validation
   - [ ] Regular security audits

2. **Performance Improvements**
   - [ ] Optimize database queries
   - [ ] Implement connection pooling
   - [ ] Add response caching
   - [ ] Reduce memory footprint

3. **Bug Fixes**
   - [ ] Fix subscription update race conditions
   - [ ] Resolve transparent proxy leaks on crash
   - [ ] Improve error handling consistency
   - [ ] Fix platform-specific edge cases

### Completed Features

- ✅ **Load Balancing (Round Robin)** - Global toggle in Settings to distribute connections evenly across active servers using round-robin algorithm
  - Basic round-robin implementation
  - Dynamic server health checking integration
  - Single-node support

### Advanced Options (Future Work)

- [ ] **Advanced Load Balancing Algorithms**
  - [ ] Weighted round-robin
  - [ ] Least connections
  - [ ] IP hash / session persistence
  - [ ] Latency-based selection
  - [ ] Custom weighting per server

- [ ] **Multi-node Load Balancing**
  - [ ] Distributed load balancing across multiple v2rayA instances
  - [ ] Centralized configuration management
  - [ ] Health check coordination

- [ ] **Load Balancing Analytics**
  - [ ] Traffic distribution statistics
  - [ ] Server performance metrics
  - [ ] Load balancing effectiveness reports

### Medium Priority

1. **UI/UX Improvements**
   - [ ] Responsive design enhancements
   - [ ] Dark mode support
   - [ ] Improved mobile experience
   - [ ] Accessibility improvements (WCAG 2.1)

2. **Monitoring & Logging**
   - [ ] Structured logging with levels
   - [ ] Traffic statistics dashboard
   - [ ] Connection history
   - [ ] Export logs feature

3. **Configuration Management**
   - [ ] Configuration import/export
   - [ ] Profile management
   - [ ] Backup and restore
   - [ ] Cloud sync (optional, user-controlled)

## Medium-term Roadmap (6-12 months)

### Major Features

1. **Plugin System**
   - [ ] Plugin architecture design
   - [ ] Plugin SDK/CLI
   - [ ] Plugin marketplace
   - [ ] Community plugin support

2. **Advanced Routing**
   - [ ] Visual routing rule editor
   - [ ] Rule testing and simulation
   - [ ] Rule sharing and import
   - [ ] Machine learning-based routing

3. **Multi-user Support**
   - [ ] Role-based access control
   - [ ] User management interface
   - [ ] Usage quotas and limits
   - [ ] Audit logging

4. **Network Features**
   - [ ] WireGuard integration
   - [ ] TUN mode support
   - [ ] Split tunneling improvements
   - [ ] DNS-over-HTTPS/TLS enhancements

5. **Live Flow Visualization**
   - [ ] Real-time animated dashboard of proxy traffic flows (source → proxy → destination)
   - [ ] Uses WebSocket for efficient, low-latency updates
   - [ ] Dynamic, interactive frontend visualization in Vue.js
   - [ ] Extensible foundation for advanced filtering and collaborative features
   - 📋 **Implementation Guide:** [LIVE_FLOW_IMPLEMENTATION.md](LIVE_FLOW_IMPLEMENTATION.md)

### Platform Expansion

1. **Mobile Support**
   - [ ] Android companion app
   - [ ] iOS integration
   - [ ] Mobile-optimized GUI

2. **Embedded Systems**
   - [ ] OpenWrt optimization
   - [ ] Router firmware integration
   - [ ] Low-memory optimizations

3. **Cloud Integration**
   - [ ] Kubernetes operator
   - [ ] Cloud deployment templates
   - [ ] Infrastructure-as-code support

## Long-term Vision (1-2 years)

### Ecosystem Development

1. **Community Platform**
   - [ ] Official plugin repository
   - [ ] Community forum
   - [ ] Documentation site
   - [ ] Tutorial and guide library

2. **Enterprise Features**
   - [ ] Centralized management console
   - [ ] LDAP/Active Directory integration
   - [ ] Compliance and audit features
   - [ ] SLA and monitoring integration

3. **AI/ML Integration**
   - [ ] Intelligent routing optimization
   - [ ] Anomaly detection
   - [ ] Predictive scaling
   - [ ] Natural language configuration

## Technical Debt

### Code Quality

- [ ] Increase test coverage to 80%
- [ ] Refactor legacy code sections
- [ ] Improve error handling patterns
- [ ] Add comprehensive documentation

### Architecture

- [ ] Implement clean architecture patterns
- [ ] Improve separation of concerns
- [ ] Add dependency injection container
- [ ] Implement CQRS for complex operations

### Dependencies

- [ ] Regular dependency updates
- [ ] Remove unused dependencies
- [ ] Evaluate alternative libraries
- [ ] Improve build system

## Known Issues

### Critical

- None reported currently

### High

1. **Transparent Proxy Leaks**
   - Some traffic may leak on unexpected shutdown
   - Workaround: Manual cleanup
   - Status: Investigation ongoing

2. **Memory Usage**
   - Long-running sessions may accumulate memory
   - Workaround: Periodic restarts
   - Status: Optimization planned

### Medium

1. **Subscription Updates**
   - Large subscriptions may timeout
   - Workaround: Split subscriptions
   - Status: Batch processing implemented

2. **Platform-specific Issues**
   - Some features unavailable on Windows/macOS
   - Workaround: Use Linux for full features
   - Status: Feature parity planned

### Low

1. **UI Responsiveness**
   - Large server lists may slow UI
   - Workaround: Virtual scrolling implemented
   - Status: Ongoing optimization

2. **Log Rotation**
   - Logs may grow large over time
   - Workaround: Manual cleanup
   - Status: Auto-rotation planned

## Feature Requests

### Top Community Requests

1. **Split Tunneling UI**
   - Visual interface for process-based routing
   - Priority: High
   - Status: Design phase

2. **Multi-server Load Balancing**
   - Distribute traffic across servers
   - Priority: Medium
   - Status: Evaluation

3. **Protocol Compression**
   - Reduce bandwidth usage
   - Priority: Low
   - Status: Research

### Experimental Features

1. **AI-powered Routing**
   - Machine learning for optimal routing
   - Status: Prototype

2. **Blockchain Integration**
   - Decentralized node discovery
   - Status: Research

3. **AR/VR Interface**
   - 3D visualization of network topology
   - Status: Concept

## Release Cycle

### Versioning

- **Major (X.0):** Breaking changes, new architecture
- **Minor (X.Y):** New features, backward compatible
- **Patch (X.Y.Z):** Bug fixes, security updates

### Release Schedule

- **Major:** Every 12-18 months
- **Minor:** Every 2-3 months
- **Patch:** As needed (critical fixes immediately)

### Long-term Support (LTS)

- **Current LTS:** v2.0.x
- **Next LTS:** v3.0.x (planned)
- **LTS Duration:** 24 months

## Contributing to Roadmap

We welcome community input on priorities!

1. **Vote on Issues:** Add 👍 to GitHub issues
2. **Propose Features:** Open feature request issues
3. **Join Discussions:** Participate in GitHub Discussions
4. **Contribute:** Submit PRs for planned features

## Roadmap Review

This roadmap is reviewed quarterly and updated based on:
- Community feedback
- Security considerations
- Technical feasibility
- Resource availability

Last updated: 2026-08-27
