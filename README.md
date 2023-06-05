# README

##### 项目编写初始化步骤
* ``wails init -n otool -t svelte`` wails命令创建项目
* ``npm create svelte@latest frontend`` 删除原始frontend目录创建新目录，创建时选择Skeleton project
* 修改frontend/package.json中devDependencies下的@sveltejs/adapter-auto为"@sveltejs/adapter-static":"next"
* 修改frontend/svelte.config.js中adapter为"@sveltejs/adapter-static"
* ``cd frontend; npm install bootstrap@5.3.0-alpha1`` 安装bootstrap
* frontend/src/routes下面添加+layout.ts文件,输入``export const ssr = false;export const prerender = true;``
* 引入bootstrap: import 'bootstrap/dist/css/bootstrap.min.css'  import 'bootstrap/dist/js/bootstrap.bundle.min.js'
* vite.config.js中加入：
```javascript
server: {
		fs: {
			allow: ['./wailsjs/go']
		}
	}
```

##### 构建命令
wails build -clean
windows amd64: ``wails build -platform=windows/amd64``
windows arm64: ``wails build -platform=windows/arm64``
mac intel chip: ``wails build -platform=darwin/amd64``
mac M chip: ``wails build -platform=darwin/arm64``
linux amd64: ``wails build -platform=linux/amd64``
linux arm64: ``wails build -platform=linux/arm64``
