# Framework Analyzer

一个智能的框架代码分析与教学技能，专门用于分析Go和Python框架代码，帮助开发者深入理解框架架构、设计模式和业务场景。

## 🚀 快速开始

### 安装依赖
```bash
cd framework-analyzer/scripts
pip install -r requirements.txt
```

### 基础使用
```bash
# 分析Go项目
python analyzer.py --path /path/to/go/project --language go

# 交互式学习
python analyzer.py --path /path/to/project --interactive

# 生成可视化图表
python visualizer.py --path /path/to/project --type all

# 检测设计模式
python pattern_detector.py --path /path/to/project --patterns factory,singleton

# 生成教程
python tutorial_generator.py --path /path/to/project --type overview
```

## 📋 功能特性

### 🔍 智能代码分析
- **AST解析**: 深度解析Go和Python源代码结构
- **设计模式识别**: 自动识别MVC、依赖注入、工厂模式等常见模式
- **组件依赖分析**: 分析组件间的依赖关系和交互模式
- **业务场景识别**: 基于代码特征自动识别AI、微服务等业务领域

### 📚 交互式学习体验
- **引导式教程**: 通过对话方式确定学习深度和重点
- **分层学习路径**: 架构概览 → 业务场景识别 → 功能模块深入
- **阶段性文档**: 自动生成markdown格式的学习文档
- **按需深入**: 用户可选择深入分析特定模块

### 📊 可视化支持
- **架构图生成**: 自动生成Mermaid格式的系统架构图
- **依赖关系图**: 可视化组件间的复杂依赖关系
- **数据流图**: 展示系统中的数据流向和处理过程
- **智能图表选择**: 根据内容自动选择最适合的图表类型

### 🛠️ 多语言支持
- **Go语言优先**: 完整支持Go框架分析，包括并发模式和接口设计
- **Python辅助**: 支持Python框架的基础分析和特有模式识别
- **混合项目**: 处理Go和Python混合的项目结构

## 📁 项目结构

```
framework-analyzer/
├── skills.md              # 技能说明文档
├── README.md              # 项目说明
├── examples/              # 示例和教程
│   ├── basic_analysis.md
│   ├── advanced_tutorial.md
│   └── power_ai_analysis_demo.md
└── scripts/               # 核心脚本
    ├── analyzer.py        # 主分析脚本
    ├── visualizer.py      # 可视化生成器
    ├── pattern_detector.py # 设计模式检测器
    ├── tutorial_generator.py # 教程生成器
    ├── requirements.txt   # Python依赖
    └── config.yaml       # 配置文件
```

## 🎯 使用场景

### 1. 新项目学习
- 快速理解新框架的架构设计
- 学习最佳实践和设计模式
- 生成学习文档和教程

### 2. 代码审查
- 分析代码质量和架构合理性
- 识别潜在的设计问题
- 生成架构文档

### 3. 团队协作
- 帮助新成员快速上手
- 统一团队对架构的理解
- 建立开发规范

### 4. 架构演进
- 跟踪架构变化
- 评估重构效果
- 制定改进计划

## 📖 详细文档

### 核心脚本说明

#### analyzer.py - 主分析脚本
```bash
# 基础分析
python analyzer.py --path /path/to/project

# 指定语言
python analyzer.py --path /path/to/project --language go

# 交互式模式
python analyzer.py --path /path/to/project --interactive

# 输出到文件
python analyzer.py --path /path/to/project --output analysis.json

# 详细输出
python analyzer.py --path /path/to/project --verbose
```

#### visualizer.py - 可视化生成器
```bash
# 生成架构图
python visualizer.py --path /path/to/project --type architecture

# 生成所有图表
python visualizer.py --path /path/to/project --type all

# 指定输出目录
python visualizer.py --path /path/to/project --output ./diagrams/
```

#### pattern_detector.py - 设计模式检测器
```bash
# 检测所有模式
python pattern_detector.py --path /path/to/project

# 检测特定模式
python pattern_detector.py --path /path/to/project --patterns factory,singleton,observer

# 输出详细报告
python pattern_detector.py --path /path/to/project --output pattern_report.md --verbose
```

#### tutorial_generator.py - 教程生成器
```bash
# 生成架构概览教程
python tutorial_generator.py --path /path/to/project --type overview

# 生成模块深入教程
python tutorial_generator.py --path /path/to/project --type module --module AgentApp

# 生成所有教程
python tutorial_generator.py --path /path/to/project --type all
```

### 配置文件说明

`config.yaml` 文件包含了所有可配置的选项：

```yaml
# 分析配置
analysis:
  max_file_size: "10MB"
  supported_extensions: [".go", ".py"]
  exclude_dirs: ["vendor", "node_modules", ".git"]

# 可视化配置
visualization:
  default_format: "mermaid"
  max_nodes: 50

# 学习配置
learning:
  default_depth: "intermediate"
  interaction_timeout: 300
```

## 🎨 支持的设计模式

### Go语言模式
- **创建型模式**: Factory, Builder, Singleton
- **结构型模式**: Adapter, Decorator, Facade
- **行为型模式**: Observer, Strategy, Command
- **并发模式**: Worker Pool, Pipeline, Fan-out/Fan-in
- **架构模式**: Dependency Injection, Repository, MVC

### Python模式
- **创建型模式**: Factory, Singleton
- **结构型模式**: Decorator, Adapter
- **行为型模式**: Observer, Strategy
- **架构模式**: MVC, Repository

## 🏗️ 支持的业务场景

### AI应用框架
- 机器学习平台
- 向量搜索系统
- 知识图谱应用
- RAG系统

### 微服务架构
- 服务网格
- API网关
- 服务发现
- 配置管理

### Web应用框架
- RESTful API
- MVC架构
- 中间件系统
- 模板引擎

### 数据处理框架
- ETL管道
- 流处理
- 批处理
- 数据分析

## 🔧 扩展开发

### 添加新的设计模式
1. 在相应的检测器中添加模式定义
2. 定义模式指示器和描述
3. 实现模式匹配逻辑
4. 添加测试用例

### 添加新的业务场景
1. 在配置文件中添加关键词
2. 实现场景识别逻辑
3. 添加场景特定的建议
4. 更新可视化模板

### 添加新的可视化类型
1. 在visualizer.py中添加新的图表类型
2. 实现图表生成逻辑
3. 添加相应的模板
4. 更新配置文件

## 🧪 测试

### 运行测试
```bash
# 运行所有测试
python -m pytest tests/

# 运行特定测试
python -m pytest tests/test_analyzer.py

# 生成覆盖率报告
python -m pytest --cov=scripts tests/
```

### 测试用例
- 单元测试：测试各个组件的功能
- 集成测试：测试完整的分析流程
- 性能测试：测试大型项目的处理能力
- 边界测试：测试异常情况的处理

## 📊 性能优化

### 大型项目处理
- 并行文件处理
- 内存使用优化
- 缓存机制
- 增量分析

### 配置建议
```yaml
performance:
  max_workers: 4
  max_memory_usage: "1GB"
  cache:
    enabled: true
    max_size: 1000
    ttl: 3600
```

## 🤝 贡献指南

### 开发环境设置
```bash
# 克隆项目
git clone <repository-url>
cd framework-analyzer

# 安装开发依赖
pip install -r scripts/requirements.txt
pip install -r requirements-dev.txt

# 运行测试
python -m pytest tests/
```

### 提交规范
- 遵循PEP 8代码规范
- 添加适当的测试用例
- 更新相关文档
- 提交前运行所有测试

## 📄 许可证

MIT License - 详见 LICENSE 文件

## 🆘 支持和反馈

- 📧 Email: support@framework-analyzer.com
- 🐛 Issues: GitHub Issues
- 💬 讨论: GitHub Discussions
- 📖 文档: https://docs.framework-analyzer.com

## 🎉 致谢

感谢所有贡献者和使用者的支持！

---

**Framework Analyzer** - 让框架学习变得简单高效！