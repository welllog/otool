# README

##### 项目编写初始化步骤

- `wails init -n otool -t svelte` wails 命令创建项目
- `npm create svelte@latest frontend` 删除原始 frontend 目录创建新目录，创建时选择 Skeleton project
- 修改 frontend/package.json 中 devDependencies 下的@sveltejs/adapter-auto 为"@sveltejs/adapter-static":"next"
- 修改 frontend/svelte.config.js 中 adapter 为"@sveltejs/adapter-static"
- `cd frontend; npm install bootstrap@5.3.0-alpha1` 安装 bootstrap
- frontend/src/routes 下面添加+layout.ts 文件,输入`export const ssr = false;export const prerender = true;`
- 引入 bootstrap: import 'bootstrap/dist/css/bootstrap.min.css' import 'bootstrap/dist/js/bootstrap.bundle.min.js'
- `npm install @types/node`
- vite.config.js 中加入：

```javascript
import path from "path";
export default defineConfig({
  plugins: [sveltekit()],
  resolve: {
    alias: {
      $wailsjs: path.resolve(__dirname, "./wailsjs"),
    },
  },
  server: {
    fs: {
      allow: ["./wailsjs"],
    },
  },
});
```

##### 构建命令

* wails build -clean
* windows amd64: `wails build -platform=windows/amd64`
* windows arm64: `wails build -platform=windows/arm64`
* mac: `wails build -platform=darwin`
* mac intel chip: `wails build -platform=darwin/amd64`
* mac M chip: `wails build -platform=darwin/arm64`
* linux amd64: `wails build -platform=linux/amd64`
* linux arm64: `wails build -platform=linux/arm64`
