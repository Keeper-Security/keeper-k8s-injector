# Keeper K8s Injector - Documentation

**Start here** if you're new to Keeper K8s Injector.

## 🚀 Getting Started (Start Here!)

New to Keeper K8s Injector? Follow these in order:

1. **[Overview](overview.md)** (3 min) - What is this and why use it?
2. **[Installation](installation.md)** (5 min) - Get it running in your cluster
3. **Try an example**: [Hello Secrets](../examples/01-hello-secrets/)

## 📖 Configuration

Once you're running, configure for your needs:

- **[Configuration Guide](configuration.md)** - Complete reference for all annotations and Helm values

## 🎯 Usage Guides

Learn how to use specific features:

- **[Injection Modes](injection-modes.md)** - Files, Environment Variables, K8s Secrets
- **[Templates](templates.md)** - Advanced secret formatting with Go templates
- **[Secret Rotation](rotation.md)** - Automatic updates with sidecar
- **[Troubleshooting](troubleshooting.md)** - Fix common problems

## 🏗️ Advanced Topics

Production deployment and enterprise features:

- **[Production Deployment](production.md)** - HA, monitoring, GitOps
- **[Architecture Deep Dive](architecture.md)** - How it works internally
- **[Cloud Authentication](cloud-auth.md)** - AWS Secrets Manager, GCP, Azure Key Vault
- **[Corporate Proxies](corporate-proxy.md)** - SSL inspection (Zscaler, Palo Alto, Cisco)

## 📚 Reference

- **[Migration Guide](migration.md)** - Coming from External Secrets Operator or Vault
- **[Comparison](comparison.md)** - vs ESO, Vault, 1Password, AWS CSI

## 💬 Need Help?

- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions
- **[GitHub Issues](https://github.com/Keeper-Security/keeper-k8s-injector/issues)** - Report bugs or request features
- **[Keeper Support](https://www.keepersecurity.com/support.html)** - Enterprise support

## 📝 For Developers

- [TESTING.md](../TESTING.md) - Running tests
- [RELEASING.md](../RELEASING.md) - Release procedures
