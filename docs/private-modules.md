# 私有依赖配置

拉取 `nexis.run` 私有模块前，需要完成以下一次性配置。

## 1. 使用 SSH 替换 HTTPS

```bash
git config --global url."git@gitlab.liasica.com:".insteadOf "https://gitlab.liasica.com/"
```

## 2. 设置 Go 私有模块环境变量

推荐使用 `go-private` 脚本（安装见下节），一行同时配置 `GOPRIVATE`、`GONOPROXY`、`GONOSUMDB` 三个变量：

```bash
go-private add nexis.run
```

若不想装脚本，也可以只设置 `GOPRIVATE`（`GONOPROXY` 与 `GONOSUMDB` 默认取其值，但会覆盖已有配置）：

```bash
go env -w GOPRIVATE=nexis.run
```

### 安装 go-private 脚本

`go-private` 会同时向三个变量追加值（自动去重、保留已有值），并记录添加历史，支持一键清空。仅需安装一次：

```bash
sudo tee /usr/local/bin/go-private >/dev/null <<'EOF'
#!/usr/bin/env bash
# Go 私有模块环境变量管理，同时作用于 GOPRIVATE、GONOPROXY、GONOSUMDB
# 用法：
#   go-private add <VALUE>   追加值（自动去重、保留已有值）
#   go-private clear         一键清空本脚本添加过的内容（不影响原有值）
#   go-private list          查看当前值
set -euo pipefail

KEYS=(GOPRIVATE GONOPROXY GONOSUMDB)
STATE_FILE="${XDG_STATE_HOME:-$HOME/.local/state}/go-private/added"

usage() {
  echo "用法：go-private add <VALUE> | clear | list" >&2
  exit 1
}

show() {
  for k in "${KEYS[@]}"; do
    echo "$k=$(go env "$k")"
  done
}

case "${1:-}" in
add)
  [[ $# -eq 2 && -n "$2" ]] || usage
  mkdir -p "$(dirname "$STATE_FILE")"
  for k in "${KEYS[@]}"; do
    cur=$(go env "$k")
    # 已存在的值不重复添加，也不记入清理清单
    case ",$cur," in *",$2,"*) continue ;; esac
    merged=$({ tr ',' '\n' <<<"$cur"; echo "$2"; } | grep -v '^$' | sort -u | paste -sd, -)
    go env -w "$k=$merged"
    echo "$k $2" >>"$STATE_FILE"
  done
  sort -u "$STATE_FILE" -o "$STATE_FILE" 2>/dev/null || true
  show
  ;;
clear)
  [[ -s $STATE_FILE ]] || { echo "没有本脚本添加的记录，无需清理"; exit 0; }
  for k in "${KEYS[@]}"; do
    added=$(awk -v k="$k" '$1 == k { print $2 }' "$STATE_FILE")
    [[ -n $added ]] || continue
    remain=$(go env "$k" | tr ',' '\n' | grep -v '^$' | grep -vxF -f <(echo "$added") | paste -sd, - || true)
    if [[ -n $remain ]]; then
      go env -w "$k=$remain"
    else
      go env -u "$k"
    fi
  done
  rm -f "$STATE_FILE"
  show
  ;;
list)
  show
  ;;
*)
  usage
  ;;
esac
EOF
sudo chmod +x /usr/local/bin/go-private
```

### 常用命令

```bash
# 追加（同时作用于三个变量，重复执行不会产生重复值）
go-private add nexis.run

# 查看三个变量的当前值
go-private list

# 一键清空本脚本添加过的内容（安装脚本前已有的值不受影响）
go-private clear
```

> 若此前安装过旧版 `append-go-env`，可执行 `sudo rm -f /usr/local/bin/append-go-env` 删除。

## 3. 安装依赖

```bash
go get -u -v nexis.run/nexa
```
