#!/usr/bin/env python3
"""
Tutorial Generator - 教程生成器

功能:
1. 生成架构概览教程
2. 生成模块深入分析教程
3. 生成最佳实践指南
4. 生成完整学习路径
"""

import os
import argparse
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from datetime import datetime
import json

@dataclass
class TutorialSection:
    """教程章节"""
    title: str
    content: str
    code_examples: List[str]
    diagrams: List[str]
    next_steps: List[str]

@dataclass
class Tutorial:
    """完整教程"""
    title: str
    description: str
    sections: List[TutorialSection]
    prerequisites: List[str]
    learning_objectives: List[str]
    estimated_time: str

class TutorialGenerator:
    """教程生成器"""
    
    def __init__(self):
        self.templates = {
            'architecture_overview': self._get_architecture_template(),
            'module_deep_dive': self._get_module_template(),
            'best_practices': self._get_best_practices_template(),
            'getting_started': self._get_getting_started_template()
        }
    
    def generate_architecture_overview(self, analysis) -> Tutorial:
        """生成架构概览教程"""
        
        project_name = Path(analysis.structure.root_path).name
        
        # 架构概览章节
        overview_section = TutorialSection(
            title="整体架构概览",
            content=self._generate_architecture_content(analysis),
            code_examples=self._extract_architecture_examples(analysis),
            diagrams=["architecture_diagram.mermaid"],
            next_steps=["了解核心组件", "学习设计模式", "深入业务逻辑"]
        )
        
        # 技术栈章节
        tech_stack_section = TutorialSection(
            title="技术栈分析",
            content=self._generate_tech_stack_content(analysis),
            code_examples=self._extract_dependency_examples(analysis),
            diagrams=["dependency_graph.mermaid"],
            next_steps=["配置开发环境", "理解依赖关系"]
        )
        
        # 核心组件章节
        components_section = TutorialSection(
            title="核心组件介绍",
            content=self._generate_components_content(analysis),
            code_examples=self._extract_component_examples(analysis),
            diagrams=["component_diagram.mermaid"],
            next_steps=["深入组件实现", "学习组件交互"]
        )
        
        return Tutorial(
            title=f"{project_name} 架构概览教程",
            description=f"深入理解 {project_name} 框架的整体架构设计",
            sections=[overview_section, tech_stack_section, components_section],
            prerequisites=["基础编程知识", f"{analysis.language.value.title()}语言基础"],
            learning_objectives=[
                "理解框架整体架构",
                "掌握核心组件功能",
                "了解技术选型原因",
                "学会架构分析方法"
            ],
            estimated_time="30-45分钟"
        )
    
    def generate_module_deep_dive(self, analysis, module_name: str) -> Tutorial:
        """生成模块深入分析教程"""
        
        # 查找指定模块
        module_info = self._find_module_info(analysis, module_name)
        
        # 模块概述章节
        overview_section = TutorialSection(
            title=f"{module_name} 模块概述",
            content=self._generate_module_overview(module_info, analysis),
            code_examples=self._extract_module_examples(module_info),
            diagrams=[f"{module_name.lower()}_structure.mermaid"],
            next_steps=["理解模块职责", "学习接口设计"]
        )
        
        # 实现细节章节
        implementation_section = TutorialSection(
            title="实现细节分析",
            content=self._generate_implementation_analysis(module_info, analysis),
            code_examples=self._extract_implementation_examples(module_info),
            diagrams=[f"{module_name.lower()}_flow.mermaid"],
            next_steps=["实践代码修改", "扩展模块功能"]
        )
        
        # 使用示例章节
        usage_section = TutorialSection(
            title="使用示例",
            content=self._generate_usage_examples(module_info, analysis),
            code_examples=self._extract_usage_examples(module_info),
            diagrams=[],
            next_steps=["自定义配置", "集成其他模块"]
        )
        
        return Tutorial(
            title=f"{module_name} 模块深入分析",
            description=f"深入学习 {module_name} 模块的设计与实现",
            sections=[overview_section, implementation_section, usage_section],
            prerequisites=["框架基础知识", "架构概览理解"],
            learning_objectives=[
                f"掌握{module_name}模块功能",
                "理解模块设计原理",
                "学会模块使用方法",
                "能够扩展模块功能"
            ],
            estimated_time="45-60分钟"
        )
    
    def generate_best_practices_guide(self, analysis) -> Tutorial:
        """生成最佳实践指南"""
        
        # 代码规范章节
        coding_section = TutorialSection(
            title="代码规范与风格",
            content=self._generate_coding_standards(analysis),
            code_examples=self._extract_style_examples(analysis),
            diagrams=[],
            next_steps=["配置代码检查工具", "建立团队规范"]
        )
        
        # 架构模式章节
        patterns_section = TutorialSection(
            title="架构模式应用",
            content=self._generate_patterns_guide(analysis),
            code_examples=self._extract_pattern_examples(analysis),
            diagrams=["patterns_overview.mermaid"],
            next_steps=["实践设计模式", "重构现有代码"]
        )
        
        # 性能优化章节
        performance_section = TutorialSection(
            title="性能优化建议",
            content=self._generate_performance_guide(analysis),
            code_examples=self._extract_performance_examples(analysis),
            diagrams=["performance_optimization.mermaid"],
            next_steps=["性能测试", "监控指标设置"]
        )
        
        # 部署运维章节
        deployment_section = TutorialSection(
            title="部署与运维",
            content=self._generate_deployment_guide(analysis),
            code_examples=self._extract_deployment_examples(analysis),
            diagrams=["deployment_architecture.mermaid"],
            next_steps=["环境配置", "监控告警设置"]
        )
        
        return Tutorial(
            title="框架开发最佳实践",
            description="基于框架分析的开发、部署和运维最佳实践",
            sections=[coding_section, patterns_section, performance_section, deployment_section],
            prerequisites=["框架深度理解", "生产环境经验"],
            learning_objectives=[
                "掌握代码规范",
                "应用设计模式",
                "优化系统性能",
                "实现可靠部署"
            ],
            estimated_time="60-90分钟"
        )
    
    def generate_getting_started_guide(self, analysis) -> Tutorial:
        """生成快速入门指南"""
        
        # 环境准备章节
        setup_section = TutorialSection(
            title="环境准备",
            content=self._generate_setup_content(analysis),
            code_examples=self._extract_setup_examples(analysis),
            diagrams=[],
            next_steps=["安装依赖", "配置开发环境"]
        )
        
        # 快速开始章节
        quickstart_section = TutorialSection(
            title="快速开始",
            content=self._generate_quickstart_content(analysis),
            code_examples=self._extract_quickstart_examples(analysis),
            diagrams=["quickstart_flow.mermaid"],
            next_steps=["运行示例", "修改配置"]
        )
        
        # 基础概念章节
        concepts_section = TutorialSection(
            title="基础概念",
            content=self._generate_concepts_content(analysis),
            code_examples=self._extract_concept_examples(analysis),
            diagrams=["concepts_overview.mermaid"],
            next_steps=["深入学习", "实践项目"]
        )
        
        return Tutorial(
            title="框架快速入门指南",
            description="帮助新手快速上手框架开发",
            sections=[setup_section, quickstart_section, concepts_section],
            prerequisites=["基础编程知识"],
            learning_objectives=[
                "搭建开发环境",
                "运行第一个示例",
                "理解核心概念",
                "开始实际开发"
            ],
            estimated_time="20-30分钟"
        )
    
    def _generate_architecture_content(self, analysis) -> str:
        """生成架构内容"""
        content = f"""
## 架构设计理念

{Path(analysis.structure.root_path).name} 采用了现代化的{analysis.scenario.domain}架构设计，主要特点包括：

### 🏗️ 分层架构
框架采用清晰的分层架构，每层都有明确的职责：

"""
        
        if analysis.language.value == 'go':
            content += """
- **应用层**: 负责应用程序的生命周期管理和配置
- **服务层**: 提供HTTP服务和API接口
- **业务层**: 实现核心业务逻辑和规则
- **数据层**: 处理数据存储和访问
- **基础设施层**: 提供日志、监控等基础服务

### 🎯 设计原则
"""
        
        # 根据检测到的模式添加设计原则
        if analysis.patterns:
            content += "框架遵循以下设计原则：\n\n"
            for pattern in analysis.patterns[:3]:
                content += f"- **{pattern.name}**: {pattern.description}\n"
        
        content += f"""

### 🔧 技术选型
基于业务需求，框架选择了以下技术栈：

- **主要语言**: {analysis.language.value.title()}
- **业务领域**: {analysis.scenario.domain}
- **核心组件**: {len(analysis.components)} 个主要组件
"""
        
        if analysis.scenario.middleware:
            content += f"- **中间件**: {', '.join(analysis.scenario.middleware)}\n"
        
        return content
    
    def _generate_tech_stack_content(self, analysis) -> str:
        """生成技术栈内容"""
        content = """
## 技术栈详解

### 核心依赖分析
"""
        
        if analysis.dependencies:
            content += "框架的主要依赖包括：\n\n"
            for i, dep in enumerate(analysis.dependencies[:10], 1):
                # 尝试识别依赖类型
                dep_type = self._identify_dependency_type(dep)
                content += f"{i}. **{dep}** - {dep_type}\n"
        
        content += """

### 中间件集成
"""
        
        if analysis.scenario.middleware:
            content += "框架集成了以下中间件：\n\n"
            middleware_descriptions = {
                'Database': '关系型数据库，用于持久化存储',
                'Cache': '缓存系统，提高数据访问性能',
                'Vector Database': '向量数据库，支持AI应用的向量检索',
                'Knowledge Graph': '知识图谱，支持复杂关系查询',
                'Object Storage': '对象存储，处理文件和媒体资源',
                'Service Discovery': '服务发现，支持微服务架构'
            }
            
            for mw in analysis.scenario.middleware:
                desc = middleware_descriptions.get(mw, '专用中间件组件')
                content += f"- **{mw}**: {desc}\n"
        
        return content
    
    def _generate_components_content(self, analysis) -> str:
        """生成组件内容"""
        content = """
## 核心组件介绍

### 组件架构
框架采用模块化的组件设计，主要组件包括：

"""
        
        for i, component in enumerate(analysis.components[:8], 1):
            # 推断组件功能
            component_desc = self._infer_component_description(component, analysis)
            content += f"### {i}. {component.replace('_', ' ').title()}\n"
            content += f"{component_desc}\n\n"
        
        if len(analysis.components) > 8:
            content += f"... 还有 {len(analysis.components) - 8} 个其他组件\n\n"
        
        content += """
### 组件交互
组件之间通过明确定义的接口进行交互，确保：
- 低耦合：组件间依赖最小化
- 高内聚：组件内部功能紧密相关
- 可测试：每个组件都可以独立测试
- 可扩展：新组件可以轻松集成
"""
        
        return content
    
    def _identify_dependency_type(self, dependency: str) -> str:
        """识别依赖类型"""
        dep_lower = dependency.lower()
        
        if any(db in dep_lower for db in ['postgres', 'mysql', 'sqlite', 'mongo']):
            return '数据库驱动'
        elif any(web in dep_lower for web in ['gin', 'echo', 'fiber', 'http']):
            return 'Web框架'
        elif any(cache in dep_lower for cache in ['redis', 'memcache']):
            return '缓存系统'
        elif any(mq in dep_lower for mq in ['kafka', 'rabbitmq', 'nats']):
            return '消息队列'
        elif any(log in dep_lower for log in ['log', 'zap', 'logrus']):
            return '日志系统'
        elif any(test in dep_lower for test in ['test', 'mock', 'assert']):
            return '测试工具'
        elif any(ai in dep_lower for ai in ['milvus', 'weaviate', 'vector']):
            return 'AI/向量数据库'
        else:
            return '工具库'
    
    def _infer_component_description(self, component: str, analysis) -> str:
        """推断组件描述"""
        comp_lower = component.lower()
        
        if 'middleware' in comp_lower:
            return "中间件组件，提供请求处理和服务集成功能"
        elif 'server' in comp_lower or 'http' in comp_lower:
            return "服务器组件，负责HTTP请求处理和路由管理"
        elif 'config' in comp_lower:
            return "配置管理组件，处理应用配置和环境变量"
        elif 'client' in comp_lower:
            return "客户端组件，负责与外部服务的通信"
        elif 'pkg' in comp_lower or 'util' in comp_lower:
            return "工具包组件，提供通用的工具函数和辅助功能"
        elif 'tool' in comp_lower:
            return "工具组件，提供开发和运维相关的工具"
        else:
            return f"业务组件，实现{analysis.scenario.domain}相关的核心功能"
    
    def _extract_architecture_examples(self, analysis) -> List[str]:
        """提取架构示例代码"""
        examples = []
        
        if analysis.language.value == 'go':
            examples.append("""
// 应用程序主结构
type AgentApp struct {
    Manifest    *Manifest
    HttpServer  *server.HttpServer
    OnShutdown  func(ctx context.Context)
    // 中间件组件
    etcd        *etcd_mw.Etcd
    pgsql       *pgsql_mw.PgSql
    redis       *redis_mw.Redis
}
""")
            
            examples.append("""
// 工厂模式创建应用
func NewAgent(manifest string, opts ...Option) (*AgentApp, error) {
    mf, err := initManifest(manifest)
    if err != nil {
        return nil, err
    }
    
    newOpts := newOptions(opts)
    // 初始化各个组件...
    
    return &AgentApp{
        Manifest:   mf,
        HttpServer: server.New(),
        // ...
    }, nil
}
""")
        
        return examples
    
    def _extract_dependency_examples(self, analysis) -> List[str]:
        """提取依赖示例"""
        examples = []
        
        if analysis.language.value == 'go':
            examples.append("""
// go.mod 依赖管理
module your-framework

go 1.19

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/go-redis/redis/v8 v8.11.5
    // 其他依赖...
)
""")
        
        return examples
    
    def _extract_component_examples(self, analysis) -> List[str]:
        """提取组件示例"""
        examples = []
        
        # 根据组件生成示例
        for component in analysis.components[:3]:
            if analysis.language.value == 'go':
                examples.append(f"""
// {component.replace('_', ' ').title()} 组件示例
type {component.title().replace('_', '')} struct {{
    config *Config
    logger *Logger
}}

func New{component.title().replace('_', '')}(config *Config) *{component.title().replace('_', '')} {{
    return &{component.title().replace('_', '')}{{
        config: config,
        logger: NewLogger(),
    }}
}}
""")
        
        return examples
    
    def _find_module_info(self, analysis, module_name: str) -> Dict:
        """查找模块信息"""
        # 这里应该根据实际的模块分析结果返回
        return {
            'name': module_name,
            'files': [],
            'functions': [],
            'types': [],
            'interfaces': []
        }
    
    def _generate_module_overview(self, module_info: Dict, analysis) -> str:
        """生成模块概述"""
        return f"""
## {module_info['name']} 模块概述

### 模块职责
{module_info['name']} 模块是框架的核心组件之一，主要负责：

- 功能1：具体功能描述
- 功能2：具体功能描述
- 功能3：具体功能描述

### 模块结构
模块采用清晰的内部结构设计，包含以下主要部分：

- **接口定义**：定义模块对外提供的接口
- **核心实现**：实现模块的主要业务逻辑
- **配置管理**：处理模块相关的配置
- **错误处理**：统一的错误处理机制
"""
    
    def _extract_module_examples(self, module_info: Dict) -> List[str]:
        """提取模块示例"""
        return [f"// {module_info['name']} 模块示例代码"]
    
    def _generate_implementation_analysis(self, module_info: Dict, analysis) -> str:
        """生成实现分析"""
        return f"""
## {module_info['name']} 实现分析

### 核心算法
模块采用了以下核心算法和数据结构：

### 性能考虑
在实现过程中，特别注意了以下性能优化：

### 扩展点
模块提供了以下扩展点，支持自定义功能：
"""
    
    def _extract_implementation_examples(self, module_info: Dict) -> List[str]:
        """提取实现示例"""
        return [f"// {module_info['name']} 实现示例"]
    
    def _generate_usage_examples(self, module_info: Dict, analysis) -> str:
        """生成使用示例"""
        return f"""
## {module_info['name']} 使用示例

### 基础用法
以下是模块的基本使用方法：

### 高级用法
对于复杂场景，可以使用以下高级功能：

### 配置选项
模块支持以下配置选项：
"""
    
    def _extract_usage_examples(self, module_info: Dict) -> List[str]:
        """提取使用示例"""
        return [f"// {module_info['name']} 使用示例"]
    
    def _generate_coding_standards(self, analysis) -> str:
        """生成代码规范"""
        return """
## 代码规范与风格

### 命名规范
- 使用有意义的变量和函数名
- 遵循语言特定的命名约定
- 保持命名的一致性

### 代码组织
- 合理的文件和目录结构
- 清晰的模块划分
- 适当的代码注释

### 错误处理
- 统一的错误处理机制
- 详细的错误信息
- 合适的错误恢复策略
"""
    
    def _extract_style_examples(self, analysis) -> List[str]:
        """提取风格示例"""
        return ["// 代码风格示例"]
    
    def _generate_patterns_guide(self, analysis) -> str:
        """生成模式指南"""
        content = """
## 架构模式应用

### 已应用的设计模式
框架中已经应用了以下设计模式：

"""
        
        for pattern in analysis.patterns:
            content += f"#### {pattern.name}\n"
            content += f"- **类型**: {pattern.type.value}\n"
            content += f"- **描述**: {pattern.description}\n"
            content += f"- **位置**: {pattern.location}\n\n"
        
        content += """
### 推荐的模式应用
基于框架特点，建议考虑以下模式：

- **观察者模式**: 用于事件通知
- **策略模式**: 用于算法选择
- **装饰器模式**: 用于功能扩展
"""
        
        return content
    
    def _extract_pattern_examples(self, analysis) -> List[str]:
        """提取模式示例"""
        examples = []
        for pattern in analysis.patterns[:2]:
            examples.append(f"// {pattern.name} 示例\n{pattern.code_snippet}")
        return examples
    
    def _generate_performance_guide(self, analysis) -> str:
        """生成性能指南"""
        return """
## 性能优化建议

### 数据库优化
- 合理使用索引
- 优化查询语句
- 实现连接池管理

### 缓存策略
- 多级缓存设计
- 缓存失效策略
- 热点数据预加载

### 并发处理
- 合理的并发模型
- 避免锁竞争
- 异步处理优化
"""
    
    def _extract_performance_examples(self, analysis) -> List[str]:
        """提取性能示例"""
        return ["// 性能优化示例"]
    
    def _generate_deployment_guide(self, analysis) -> str:
        """生成部署指南"""
        return """
## 部署与运维

### 容器化部署
- Docker镜像构建
- Kubernetes配置
- 服务编排

### 监控告警
- 关键指标监控
- 日志收集分析
- 告警规则配置

### 运维自动化
- CI/CD流水线
- 自动化测试
- 灰度发布
"""
    
    def _extract_deployment_examples(self, analysis) -> List[str]:
        """提取部署示例"""
        return ["# Dockerfile示例", "# Kubernetes配置示例"]
    
    def _generate_setup_content(self, analysis) -> str:
        """生成环境准备内容"""
        content = f"""
## 环境准备

### 系统要求
- 操作系统：Linux/macOS/Windows
- {analysis.language.value.title()} 版本：推荐最新稳定版
"""
        
        if analysis.scenario.middleware:
            content += "- 中间件：" + "、".join(analysis.scenario.middleware) + "\n"
        
        content += """
### 开发工具
推荐使用以下开发工具：
- IDE：VS Code / GoLand / PyCharm
- 版本控制：Git
- 包管理：根据语言选择合适的包管理器
"""
        
        return content
    
    def _extract_setup_examples(self, analysis) -> List[str]:
        """提取环境准备示例"""
        examples = []
        
        if analysis.language.value == 'go':
            examples.append("""
# 安装Go
wget https://golang.org/dl/go1.19.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.19.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
""")
        elif analysis.language.value == 'python':
            examples.append("""
# 安装Python
sudo apt-get update
sudo apt-get install python3 python3-pip
pip3 install virtualenv
""")
        
        return examples
    
    def _generate_quickstart_content(self, analysis) -> str:
        """生成快速开始内容"""
        return f"""
## 快速开始

### 1. 克隆项目
```bash
git clone <repository-url>
cd {Path(analysis.structure.root_path).name}
```

### 2. 安装依赖
根据项目类型安装相应依赖

### 3. 配置环境
复制配置文件并根据需要修改

### 4. 启动服务
运行项目并验证功能
"""
    
    def _extract_quickstart_examples(self, analysis) -> List[str]:
        """提取快速开始示例"""
        examples = []
        
        if analysis.language.value == 'go':
            examples.append("""
# 安装依赖
go mod tidy

# 运行项目
go run main.go
""")
        
        return examples
    
    def _generate_concepts_content(self, analysis) -> str:
        """生成概念内容"""
        return f"""
## 基础概念

### 核心概念
理解以下核心概念对使用框架至关重要：

1. **{analysis.scenario.domain}**: 框架的主要应用领域
2. **组件化**: 模块化的组件设计
3. **配置管理**: 灵活的配置系统
4. **中间件**: 可插拔的中间件架构

### 设计理念
框架基于以下设计理念：
- 简单易用
- 高性能
- 可扩展
- 可维护
"""
    
    def _extract_concept_examples(self, analysis) -> List[str]:
        """提取概念示例"""
        return ["// 核心概念示例代码"]
    
    def save_tutorial(self, tutorial: Tutorial, output_path: str) -> str:
        """保存教程到文件"""
        
        output_file = Path(output_path)
        output_file.parent.mkdir(parents=True, exist_ok=True)
        
        # 生成Markdown内容
        content = self._tutorial_to_markdown(tutorial)
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"📚 教程已保存到: {output_file}")
        return str(output_file)
    
    def _tutorial_to_markdown(self, tutorial: Tutorial) -> str:
        """将教程转换为Markdown格式"""
        
        lines = [
            f"# {tutorial.title}\n",
            f"{tutorial.description}\n",
            f"**预计学习时间**: {tutorial.estimated_time}\n",
            f"**生成时间**: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n"
        ]
        
        # 前置条件
        if tutorial.prerequisites:
            lines.append("## 前置条件\n\n")
            for prereq in tutorial.prerequisites:
                lines.append(f"- {prereq}\n")
            lines.append("\n")
        
        # 学习目标
        if tutorial.learning_objectives:
            lines.append("## 学习目标\n\n")
            for objective in tutorial.learning_objectives:
                lines.append(f"- {objective}\n")
            lines.append("\n")
        
        # 教程章节
        for i, section in enumerate(tutorial.sections, 1):
            lines.append(f"## {i}. {section.title}\n\n")
            lines.append(f"{section.content}\n\n")
            
            # 代码示例
            if section.code_examples:
                lines.append("### 代码示例\n\n")
                for example in section.code_examples:
                    lines.append("```\n")
                    lines.append(f"{example}\n")
                    lines.append("```\n\n")
            
            # 图表
            if section.diagrams:
                lines.append("### 相关图表\n\n")
                for diagram in section.diagrams:
                    lines.append(f"- [{diagram}](./diagrams/{diagram})\n")
                lines.append("\n")
            
            # 下一步
            if section.next_steps:
                lines.append("### 下一步\n\n")
                for step in section.next_steps:
                    lines.append(f"- {step}\n")
                lines.append("\n")
        
        # 总结
        lines.append("## 总结\n\n")
        lines.append("通过本教程，您应该已经掌握了框架的核心概念和使用方法。")
        lines.append("建议继续深入学习其他相关教程，并在实际项目中应用所学知识。\n\n")
        
        # 相关资源
        lines.append("## 相关资源\n\n")
        lines.append("- [框架文档](./README.md)\n")
        lines.append("- [API参考](./api-reference.md)\n")
        lines.append("- [示例项目](./examples/)\n")
        lines.append("- [常见问题](./faq.md)\n")
        
        return ''.join(lines)
    
    def _get_architecture_template(self) -> str:
        """获取架构模板"""
        return "architecture_overview_template"
    
    def _get_module_template(self) -> str:
        """获取模块模板"""
        return "module_deep_dive_template"
    
    def _get_best_practices_template(self) -> str:
        """获取最佳实践模板"""
        return "best_practices_template"
    
    def _get_getting_started_template(self) -> str:
        """获取入门模板"""
        return "getting_started_template"

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description='Tutorial Generator - 教程生成器')
    parser.add_argument('--path', required=True, help='项目路径')
    parser.add_argument('--type', choices=['overview', 'module', 'best_practices', 'getting_started', 'all'],
                       default='overview', help='教程类型')
    parser.add_argument('--module', help='指定模块名称（用于module类型）')
    parser.add_argument('--output', help='输出目录')
    parser.add_argument('--format', choices=['markdown', 'html'], default='markdown', help='输出格式')
    
    args = parser.parse_args()
    
    try:
        # 导入分析器
        from analyzer import FrameworkAnalyzer
        
        # 分析项目
        analyzer = FrameworkAnalyzer()
        analysis = analyzer.analyze_project(args.path)
        
        # 创建教程生成器
        generator = TutorialGenerator()
        
        # 确定输出目录
        output_dir = Path(args.output) if args.output else Path(args.path) / "tutorials"
        output_dir.mkdir(exist_ok=True)
        
        # 生成教程
        if args.type == 'all':
            # 生成所有类型的教程
            tutorials = {
                'overview': generator.generate_architecture_overview(analysis),
                'getting_started': generator.generate_getting_started_guide(analysis),
                'best_practices': generator.generate_best_practices_guide(analysis)
            }
            
            for tutorial_type, tutorial in tutorials.items():
                output_file = output_dir / f"{tutorial_type}_tutorial.md"
                generator.save_tutorial(tutorial, str(output_file))
            
            print(f"✅ 所有教程已生成到: {output_dir}")
            
        else:
            # 生成指定类型的教程
            if args.type == 'overview':
                tutorial = generator.generate_architecture_overview(analysis)
                filename = "architecture_overview.md"
            elif args.type == 'module':
                if not args.module:
                    print("❌ 生成模块教程需要指定 --module 参数")
                    return
                tutorial = generator.generate_module_deep_dive(analysis, args.module)
                filename = f"{args.module}_deep_dive.md"
            elif args.type == 'best_practices':
                tutorial = generator.generate_best_practices_guide(analysis)
                filename = "best_practices.md"
            elif args.type == 'getting_started':
                tutorial = generator.generate_getting_started_guide(analysis)
                filename = "getting_started.md"
            
            output_file = output_dir / filename
            generator.save_tutorial(tutorial, str(output_file))
            
            print(f"✅ {args.type}教程已生成: {output_file}")
        
    except Exception as e:
        print(f"❌ 教程生成失败: {e}")
        import traceback
        traceback.print_exc()

if __name__ == '__main__':
    main()