# 首页与客户界面验收

首页使用白底三角网格 Logo、简洁正文与可交互三维主图。主图和三维能力卡片复用同一 WebGL Canvas；Three.js 提供平滑几何、物理材质、环境反射与 OrbitControls 阻尼，替代自绘三角面的二维渲染器。渲染依赖独立分包，模型不访问供应商。

## 交互

- 拖动带阻尼；左右按钮、方向键、Home、实体/线框及暂停均可用。纵向触摸保留页面滚动。
- 减少动态效果偏好关闭自动旋转和光晕动画；离屏、后台停止绘制，卸载释放 GPU 资源。WebGL 不可用时显示静态标志。
- 保留键盘标签导航、复制成功/失败反馈、手机菜单和 FAQ。删除装饰性英文小字、编号、伪终端标题和重复宣传区。
- 交互原则参考 Apple [Motion](https://developer.apple.com/design/human-interface-guidelines/motion) 和 [Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility)。

## 历史提交审查

两名子代理审查了 Sup3API 自定义提交与页面设计。删除失效样式和简化版 ConsoleView；恢复已有 Dashboard/Usage 完整统计，以及按权限和功能开关显示的客户导航入口。API 协议、原生兼容、分组、上游账号绑定、幂等与所有权验证均保留，未修改后端。

接入页三维模型选择目前仍用静态模型表，能力查询不会自动填充模型选项；不宣称已经实现动态模型列表。

## 验证

71 项自动化测试通过，覆盖首页导航/复制、WebGL 生命周期与降级、客户导航权限、既有用量统计、API 密钥及页面标题。命令见 [测试指南](TESTING.md)。

Chromium 实际检查 320、390、768、1440px 布局，无横向溢出，首页可见按钮至少 44×44px。真实拖动后检测到 27 帧减速绘制，停稳后停止；三维标签、减少动态效果、手机菜单、统计日期筛选与客户菜单均验证。首页/三维面板 axe 扫描分别通过 38/37 条规则，零违规。未进行实体 iPhone/Safari 验收。
