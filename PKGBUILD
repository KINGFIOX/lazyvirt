# Maintainer: KINGFIOX <https://github.com/KINGFIOX>
pkgname=lazyvirt-git
pkgver=r3.g4e75314
pkgrel=1
pkgdesc='lazygit-style TUI for managing libvirt domains, backed by virsh'
arch=('x86_64' 'aarch64')
url='https://github.com/KINGFIOX/lazyvirt'
license=('MIT')
depends=('libvirt') # provides virsh
makedepends=('git' 'go')
provides=('lazyvirt')
conflicts=('lazyvirt')
source=('git+https://github.com/KINGFIOX/lazyvirt.git')
md5sums=('SKIP')

pkgver() {
  cd lazyvirt
  printf 'r%s.g%s' "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

build() {
  cd lazyvirt
  export CGO_ENABLED=0
  export GOFLAGS='-trimpath -mod=readonly -modcacherw'
  go build -buildvcs=false -buildmode=pie \
    -ldflags "-X main.version=${pkgver}" \
    -o lazyvirt .
}

check() {
  cd lazyvirt
  export CGO_ENABLED=0
  go test ./...
}

package() {
  cd lazyvirt
  install -Dm755 lazyvirt "$pkgdir/usr/bin/lazyvirt"
  install -Dm644 README.md "$pkgdir/usr/share/doc/lazyvirt/README.md"
  install -Dm644 LICENSE "$pkgdir/usr/share/licenses/lazyvirt/LICENSE"
}
