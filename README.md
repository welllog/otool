# README

##### 项目编写初始化步骤
* ``wails init -n otool -t svelte`` wails命令创建项目
* ``npm create svelte@latest frontend`` 删除原始frontend目录创建新目录，创建时选择Skeleton project
* 修改frontend/package.json中devDependencies下的@sveltejs/adapter-auto为"@sveltejs/adapter-node":"next"
* ``cd frontend; npm install bootstrap@5.3.0-alpha1`` 安装bootstrap
* frontend/src/routes下面添加+layout.ts文件,输入``export const ssr = false;``
* 引入bootstrap: import 'bootstrap/dist/css/bootstrap.min.css'  import 'bootstrap/dist/js/bootstrap.bundle.min.js'
* vite.config.js中加入：
```javascript
server: {
		fs: {
			allow: ['./wailsjs/go/main']
		}
	}
```