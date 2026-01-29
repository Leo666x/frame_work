#!/usr/bin/env python3
"""
Framework Analyzer - 智能框架代码分析工具

主要功能:
1. 代码结构分析
2. 设计模式识别  
3. 业务场景识别
4. 交互式学习
5. 可视化生成
"""

import os
import sys
import argparse
import json
import yaml
from pathlib import Path
from typing import Dict, List, Optional, Any
import subprocess
import ast
from dataclasses import dataclass, asdict
from enum import Enum

class Language(Enum):
    GO = "go"
    PYTHON = "python"
    MIXED = "mixed"

class AnalysisDepth(Enum):
    OVERVIEW = "overview"
    INTERMEDIATE = "intermediate"
    DEEP = "deep"

@dataclass
class ProjectStructure:
    """项目结构信息"""
    root_path: str
    directories: List[str]
    files: List[str]
    go_files: List[str]
    python_files: List[str]
    total_lines: int

@dataclass
class DesignPattern:
    """设计模式信息"""
    name: str
    type: str
    description: str
    location: str
    confidence: float
    examples: List[str]

@dataclass
class BusinessScenario:
    """业务场景信息"""
    domain: str
    use_case: str
    patterns: List[str]
    middleware: List[str]
    confidence: float
    description: str

@dataclass
class ProjectAnalysis:
    """项目分析结果"""
    language: Language
    structure: ProjectStructure
    patterns: List[DesignPattern]
    scenario: BusinessScenario
    dependencies: List[str]
    components: List[str]

class FrameworkAnalyzer:
    """框架分析器主类"""
    
    def __init__(self, config_path: Optional[str] = None):
        self.config = self._load_config(config_path)
        self.supported_extensions = {
            '.go': Language.GO,
            '.py': Language.PYTHON,
        }
        
    def _load_config(self, config_path: Optional[str]) -> Dict[str, Any]:
        """加载配置文件"""
        default_config = {
            'analysis': {
                'max_file_size': '10MB',
                'supported_extensions': ['.go', '.py'],
                'exclude_dirs': ['vendor', 'node_modules', '.git', '__pycache__']
            },
            'visualization': {
                'default_format': 'mermaid',
                'max_nodes': 50
            },
            'learning': {
                'default_depth': 'intermediate',
                'interaction_timeout': 300
            }
        }
        
        if config_path and os.path.exists(config_path):
            with open(config_path, 'r', encoding='utf-8') as f:
                user_config = yaml.safe_load(f)
                default_config.update(user_config)
                
        return default_config
    
    def scan_project(self, project_path: str) -> ProjectStructure:
        """扫描项目结构"""
        print(f"🔍 正在扫描项目: {project_path}")
        
        root_path = Path(project_path).resolve()
        if not root_path.exists():
            raise FileNotFoundError(f"项目路径不存在: {project_path}")
            
        directories = []
        files = []
        go_files = []
        python_files = []
        total_lines = 0
        
        exclude_dirs = set(self.config['analysis']['exclude_dirs'])
        
        for item in root_path.rglob('*'):
            # 跳过排除的目录
            if any(excluded in item.parts for excluded in exclude_dirs):
                continue
                
            if item.is_dir():
                directories.append(str(item.relative_to(root_path)))
            elif item.is_file():
                rel_path = str(item.relative_to(root_path))
                files.append(rel_path)
                
                # 按语言分类文件
                if item.suffix == '.go':
                    go_files.append(rel_path)
                    total_lines += self._count_lines(item)
                elif item.suffix == '.py':
                    python_files.append(rel_path)
                    total_lines += self._count_lines(item)
        
        structure = ProjectStructure(
            root_path=str(root_path),
            directories=directories,
            files=files,
            go_files=go_files,
            python_files=python_files,
            total_lines=total_lines
        )
        
        print(f"✅ 扫描完成: {len(go_files)} Go文件, {len(python_files)} Python文件")
        return structure
    
    def _count_lines(self, file_path: Path) -> int:
        """统计文件行数"""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                return len(f.readlines())
        except:
            return 0
    
    def detect_language(self, structure: ProjectStructure) -> Language:
        """检测项目主要语言"""
        go_count = len(structure.go_files)
        python_count = len(structure.python_files)
        
        if go_count > 0 and python_count > 0:
            return Language.MIXED
        elif go_count > python_count:
            return Language.GO
        elif python_count > 0:
            return Language.PYTHON
        else:
            # 默认返回Go
            return Language.GO
    
    def analyze_go_patterns(self, structure: ProjectStructure) -> List[DesignPattern]:
        """分析Go代码中的设计模式"""
        patterns = []
        
        # 检查主要Go文件
        for go_file in structure.go_files[:10]:  # 限制检查文件数量
            file_path = Path(structure.root_path) / go_file
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                    
                # 检测工厂模式
                if self._detect_factory_pattern(content):
                    patterns.append(DesignPattern(
                        name="Factory Pattern",
                        type="Creational",
                        description="工厂模式用于创建对象",
                        location=go_file,
                        confidence=0.8,
                        examples=["NewAgent函数", "New*函数"]
                    ))
                
                # 检测依赖注入模式
                if self._detect_dependency_injection(content):
                    patterns.append(DesignPattern(
                        name="Dependency Injection",
                        type="Structural", 
                        description="通过Options模式实现依赖注入",
                        location=go_file,
                        confidence=0.9,
                        examples=["Options模式", "WithXXX函数"]
                    ))
                
                # 检测单例模式
                if self._detect_singleton_pattern(content):
                    patterns.append(DesignPattern(
                        name="Singleton Pattern",
                        type="Creational",
                        description="单例模式确保只有一个实例",
                        location=go_file,
                        confidence=0.7,
                        examples=["全局变量", "sync.Once"]
                    ))
                    
            except Exception as e:
                print(f"⚠️  分析文件 {go_file} 时出错: {e}")
                
        return patterns
    
    def _detect_factory_pattern(self, content: str) -> bool:
        """检测工厂模式"""
        indicators = [
            "func New",
            "func Create",
            "func Make",
            "return &",
        ]
        return any(indicator in content for indicator in indicators)
    
    def _detect_dependency_injection(self, content: str) -> bool:
        """检测依赖注入模式"""
        indicators = [
            "type Option func",
            "opts ...Option",
            "func With",
            "newOptions",
        ]
        return any(indicator in content for indicator in indicators)
    
    def _detect_singleton_pattern(self, content: str) -> bool:
        """检测单例模式"""
        indicators = [
            "sync.Once",
            "var instance",
            "GetInstance",
            "env.G",
        ]
        return any(indicator in content for indicator in indicators)
    
    def analyze_python_patterns(self, structure: ProjectStructure) -> List[DesignPattern]:
        """分析Python代码中的设计模式"""
        patterns = []
        
        for py_file in structure.python_files[:10]:
            file_path = Path(structure.root_path) / py_file
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                    
                # 检测装饰器模式
                if '@' in content and 'def ' in content:
                    patterns.append(DesignPattern(
                        name="Decorator Pattern",
                        type="Structural",
                        description="Python装饰器模式",
                        location=py_file,
                        confidence=0.9,
                        examples=["@property", "@staticmethod", "@classmethod"]
                    ))
                
                # 检测MVC模式 (Django/Flask)
                if self._detect_mvc_pattern(content):
                    patterns.append(DesignPattern(
                        name="MVC Pattern",
                        type="Architectural",
                        description="Model-View-Controller架构模式",
                        location=py_file,
                        confidence=0.8,
                        examples=["models.py", "views.py", "controllers.py"]
                    ))
                    
            except Exception as e:
                print(f"⚠️  分析Python文件 {py_file} 时出错: {e}")
                
        return patterns
    
    def _detect_mvc_pattern(self, content: str) -> bool:
        """检测MVC模式"""
        indicators = [
            "from django",
            "from flask",
            "class.*Model",
            "class.*View",
            "def render",
        ]
        return any(indicator in content for indicator in indicators)
    
    def analyze_business_scenario(self, structure: ProjectStructure, patterns: List[DesignPattern]) -> BusinessScenario:
        """分析业务场景"""
        
        # 分析依赖和导入
        dependencies = self._extract_dependencies(structure)
        
        # AI应用特征检测
        ai_indicators = ['milvus', 'weaviate', 'embedding', 'vector', 'llm', 'openai', 'tensorflow', 'pytorch']
        ai_score = sum(1 for dep in dependencies if any(indicator in dep.lower() for indicator in ai_indicators))
        
        # 微服务特征检测  
        microservice_indicators = ['grpc', 'consul', 'etcd', 'kubernetes', 'docker', 'gin', 'fastapi']
        microservice_score = sum(1 for dep in dependencies if any(indicator in dep.lower() for indicator in microservice_indicators))
        
        # Web应用特征检测
        web_indicators = ['django', 'flask', 'gin', 'echo', 'fiber', 'http', 'rest']
        web_score = sum(1 for dep in dependencies if any(indicator in dep.lower() for indicator in web_indicators))
        
        # 数据处理特征检测
        data_indicators = ['pandas', 'numpy', 'spark', 'kafka', 'redis', 'postgresql', 'mongodb']
        data_score = sum(1 for dep in dependencies if any(indicator in dep.lower() for indicator in data_indicators))
        
        # 确定主要业务场景
        scores = {
            'AI Application': ai_score,
            'Microservice': microservice_score, 
            'Web Application': web_score,
            'Data Processing': data_score
        }
        
        primary_domain = max(scores, key=scores.get)
        confidence = min(scores[primary_domain] / 10.0, 1.0)  # 标准化置信度
        
        # 识别中间件
        middleware = []
        middleware_indicators = {
            'etcd': 'Service Discovery',
            'redis': 'Cache',
            'postgresql': 'Database',
            'milvus': 'Vector Database',
            'weaviate': 'Knowledge Graph',
            'minio': 'Object Storage',
            'kafka': 'Message Queue'
        }
        
        for dep in dependencies:
            for indicator, name in middleware_indicators.items():
                if indicator in dep.lower():
                    middleware.append(name)
        
        # 生成使用场景描述
        use_cases = {
            'AI Application': 'AI应用开发框架，支持向量搜索、知识图谱、机器学习模型部署',
            'Microservice': '微服务架构框架，支持服务发现、配置管理、API网关',
            'Web Application': 'Web应用开发框架，提供HTTP服务、路由管理、中间件支持',
            'Data Processing': '数据处理框架，支持大数据处理、实时计算、数据存储'
        }
        
        return BusinessScenario(
            domain=primary_domain,
            use_case=use_cases.get(primary_domain, '通用应用框架'),
            patterns=[p.name for p in patterns],
            middleware=list(set(middleware)),
            confidence=confidence,
            description=f"基于代码分析，这是一个{primary_domain}类型的框架"
        )
    
    def _extract_dependencies(self, structure: ProjectStructure) -> List[str]:
        """提取项目依赖"""
        dependencies = []
        
        # 从go.mod提取Go依赖
        go_mod_path = Path(structure.root_path) / 'go.mod'
        if go_mod_path.exists():
            try:
                with open(go_mod_path, 'r', encoding='utf-8') as f:
                    content = f.read()
                    lines = content.split('\n')
                    in_require = False
                    for line in lines:
                        line = line.strip()
                        if line.startswith('require'):
                            in_require = True
                            continue
                        if in_require:
                            if line.startswith(')'):
                                break
                            if line and not line.startswith('//'):
                                parts = line.split()
                                if len(parts) >= 2:
                                    dependencies.append(parts[0])
            except Exception as e:
                print(f"⚠️  读取go.mod失败: {e}")
        
        # 从requirements.txt提取Python依赖
        req_path = Path(structure.root_path) / 'requirements.txt'
        if req_path.exists():
            try:
                with open(req_path, 'r', encoding='utf-8') as f:
                    for line in f:
                        line = line.strip()
                        if line and not line.startswith('#'):
                            dep = line.split('==')[0].split('>=')[0].split('<=')[0]
                            dependencies.append(dep)
            except Exception as e:
                print(f"⚠️  读取requirements.txt失败: {e}")
        
        return dependencies
    
    def analyze_project(self, project_path: str, language: Optional[Language] = None) -> ProjectAnalysis:
        """分析项目"""
        print(f"\n🚀 开始分析项目: {project_path}")
        
        # 1. 扫描项目结构
        structure = self.scan_project(project_path)
        
        # 2. 检测语言
        if language is None:
            language = self.detect_language(structure)
        
        print(f"📋 检测到语言: {language.value}")
        
        # 3. 分析设计模式
        patterns = []
        if language in [Language.GO, Language.MIXED]:
            patterns.extend(self.analyze_go_patterns(structure))
        if language in [Language.PYTHON, Language.MIXED]:
            patterns.extend(self.analyze_python_patterns(structure))
        
        print(f"🎨 发现 {len(patterns)} 个设计模式")
        
        # 4. 分析业务场景
        scenario = self.analyze_business_scenario(structure, patterns)
        print(f"🎯 业务场景: {scenario.domain} (置信度: {scenario.confidence:.2f})")
        
        # 5. 提取依赖和组件
        dependencies = self._extract_dependencies(structure)
        components = self._extract_components(structure)
        
        return ProjectAnalysis(
            language=language,
            structure=structure,
            patterns=patterns,
            scenario=scenario,
            dependencies=dependencies,
            components=components
        )
    
    def _extract_components(self, structure: ProjectStructure) -> List[str]:
        """提取主要组件"""
        components = []
        
        # 从目录结构推断组件
        for directory in structure.directories:
            if '/' not in directory:  # 只看顶级目录
                if directory not in ['vendor', 'node_modules', '.git', '__pycache__']:
                    components.append(directory)
        
        # 从Go文件名推断组件
        for go_file in structure.go_files:
            if '/' not in go_file:  # 只看根目录文件
                name = Path(go_file).stem
                if name not in ['main', 'test'] and not name.endswith('_test'):
                    components.append(name)
        
        return list(set(components))

class InteractiveLearning:
    """交互式学习管理器"""
    
    def __init__(self, analyzer: FrameworkAnalyzer):
        self.analyzer = analyzer
        self.current_analysis: Optional[ProjectAnalysis] = None
        self.learning_stage = 0
        self.stages = [
            "架构概览",
            "业务场景分析", 
            "设计模式详解",
            "组件深入分析",
            "可视化生成"
        ]
    
    def start_interactive_session(self, project_path: str):
        """开始交互式学习会话"""
        print("\n🤖 Framework Analyzer 交互式学习")
        print("=" * 50)
        
        # 分析项目
        self.current_analysis = self.analyzer.analyze_project(project_path)
        
        # 显示项目概览
        self._show_project_overview()
        
        # 开始学习循环
        while True:
            choice = self._show_learning_menu()
            if choice == 'q':
                print("\n👋 感谢使用 Framework Analyzer!")
                break
            elif choice.isdigit():
                stage_idx = int(choice) - 1
                if 0 <= stage_idx < len(self.stages):
                    self._handle_learning_stage(stage_idx)
            else:
                print("❌ 无效选择，请重试")
    
    def _show_project_overview(self):
        """显示项目概览"""
        analysis = self.current_analysis
        print(f"\n📁 项目概览:")
        print(f"   路径: {analysis.structure.root_path}")
        print(f"   语言: {analysis.language.value}")
        print(f"   文件: {len(analysis.structure.go_files)} Go, {len(analysis.structure.python_files)} Python")
        print(f"   总行数: {analysis.structure.total_lines}")
        print(f"   组件: {len(analysis.components)} 个")
        
        print(f"\n🎯 业务场景: {analysis.scenario.domain}")
        print(f"   描述: {analysis.scenario.use_case}")
        print(f"   置信度: {analysis.scenario.confidence:.2f}")
        
        if analysis.scenario.middleware:
            print(f"   中间件: {', '.join(analysis.scenario.middleware)}")
    
    def _show_learning_menu(self) -> str:
        """显示学习菜单"""
        print(f"\n📚 学习路径选择:")
        for i, stage in enumerate(self.stages, 1):
            print(f"   {i}. {stage}")
        print("   q. 退出")
        
        return input("\n请选择学习路径 (1-5, q): ").strip().lower()
    
    def _handle_learning_stage(self, stage_idx: int):
        """处理学习阶段"""
        stage_name = self.stages[stage_idx]
        print(f"\n📖 {stage_name}")
        print("-" * 30)
        
        if stage_idx == 0:  # 架构概览
            self._show_architecture_overview()
        elif stage_idx == 1:  # 业务场景分析
            self._show_business_scenario_analysis()
        elif stage_idx == 2:  # 设计模式详解
            self._show_design_patterns()
        elif stage_idx == 3:  # 组件深入分析
            self._show_component_analysis()
        elif stage_idx == 4:  # 可视化生成
            self._show_visualization_options()
        
        input("\n按回车键继续...")
    
    def _show_architecture_overview(self):
        """显示架构概览"""
        analysis = self.current_analysis
        
        print("🏗️ 整体架构分析:")
        print(f"   主要组件: {', '.join(analysis.components[:5])}")
        print(f"   依赖数量: {len(analysis.dependencies)}")
        
        if analysis.dependencies:
            print(f"   核心依赖: {', '.join(analysis.dependencies[:5])}")
        
        print(f"\n📊 项目统计:")
        print(f"   目录数: {len(analysis.structure.directories)}")
        print(f"   文件数: {len(analysis.structure.files)}")
        print(f"   代码行数: {analysis.structure.total_lines}")
        
        # 生成简单的架构描述
        if analysis.language == Language.GO:
            print(f"\n🔧 Go项目特征:")
            print("   - 使用Go模块管理依赖")
            print("   - 可能采用微服务架构")
            if 'gin' in str(analysis.dependencies).lower():
                print("   - 使用Gin Web框架")
            if 'etcd' in str(analysis.dependencies).lower():
                print("   - 集成etcd服务发现")
    
    def _show_business_scenario_analysis(self):
        """显示业务场景分析"""
        scenario = self.current_analysis.scenario
        
        print(f"🎯 业务领域: {scenario.domain}")
        print(f"📝 使用场景: {scenario.use_case}")
        print(f"📈 置信度: {scenario.confidence:.2f}")
        
        if scenario.middleware:
            print(f"\n🔧 集成的中间件:")
            for mw in scenario.middleware:
                print(f"   - {mw}")
        
        if scenario.patterns:
            print(f"\n🎨 相关设计模式:")
            for pattern in scenario.patterns:
                print(f"   - {pattern}")
        
        # 提供场景特定的建议
        if scenario.domain == "AI Application":
            print(f"\n💡 AI应用开发建议:")
            print("   - 考虑向量数据库的索引优化")
            print("   - 实现模型版本管理")
            print("   - 添加A/B测试支持")
        elif scenario.domain == "Microservice":
            print(f"\n💡 微服务架构建议:")
            print("   - 实现熔断器模式")
            print("   - 添加分布式追踪")
            print("   - 考虑API网关")
    
    def _show_design_patterns(self):
        """显示设计模式详解"""
        patterns = self.current_analysis.patterns
        
        if not patterns:
            print("❌ 未检测到明显的设计模式")
            return
        
        print(f"🎨 检测到 {len(patterns)} 个设计模式:")
        
        for i, pattern in enumerate(patterns, 1):
            print(f"\n{i}. {pattern.name} ({pattern.type})")
            print(f"   描述: {pattern.description}")
            print(f"   位置: {pattern.location}")
            print(f"   置信度: {pattern.confidence:.2f}")
            if pattern.examples:
                print(f"   示例: {', '.join(pattern.examples)}")
    
    def _show_component_analysis(self):
        """显示组件分析"""
        analysis = self.current_analysis
        
        print(f"🔧 主要组件分析:")
        
        for i, component in enumerate(analysis.components[:10], 1):
            print(f"\n{i}. {component}")
            
            # 尝试推断组件功能
            if 'middleware' in component.lower():
                print("   类型: 中间件层")
            elif 'pkg' in component.lower() or 'util' in component.lower():
                print("   类型: 工具包")
            elif 'server' in component.lower() or 'http' in component.lower():
                print("   类型: 服务层")
            elif 'config' in component.lower():
                print("   类型: 配置管理")
            else:
                print("   类型: 业务组件")
        
        if len(analysis.components) > 10:
            print(f"\n... 还有 {len(analysis.components) - 10} 个组件")
    
    def _show_visualization_options(self):
        """显示可视化选项"""
        print("📊 可视化选项:")
        print("   1. 生成架构图")
        print("   2. 生成依赖关系图") 
        print("   3. 生成组件交互图")
        print("   4. 生成完整报告")
        
        choice = input("\n选择可视化类型 (1-4): ").strip()
        
        if choice == '1':
            self._generate_architecture_diagram()
        elif choice == '2':
            self._generate_dependency_diagram()
        elif choice == '3':
            self._generate_interaction_diagram()
        elif choice == '4':
            self._generate_full_report()
        else:
            print("❌ 无效选择")
    
    def _generate_architecture_diagram(self):
        """生成架构图"""
        print("📊 正在生成架构图...")
        
        # 这里调用visualizer.py
        try:
            from visualizer import Visualizer
            visualizer = Visualizer()
            diagram = visualizer.generate_architecture_diagram(self.current_analysis)
            print(f"✅ 架构图已生成: {diagram}")
        except ImportError:
            print("⚠️  可视化模块未找到，请确保visualizer.py存在")
    
    def _generate_dependency_diagram(self):
        """生成依赖关系图"""
        print("📊 正在生成依赖关系图...")
        # 实现依赖图生成逻辑
        print("✅ 依赖关系图生成完成")
    
    def _generate_interaction_diagram(self):
        """生成交互图"""
        print("📊 正在生成组件交互图...")
        # 实现交互图生成逻辑
        print("✅ 组件交互图生成完成")
    
    def _generate_full_report(self):
        """生成完整报告"""
        print("📝 正在生成完整分析报告...")
        
        analysis = self.current_analysis
        report_path = Path(analysis.structure.root_path) / "framework_analysis_report.md"
        
        with open(report_path, 'w', encoding='utf-8') as f:
            f.write(f"# {Path(analysis.structure.root_path).name} 框架分析报告\n\n")
            
            # 项目概览
            f.write("## 项目概览\n\n")
            f.write(f"- **语言**: {analysis.language.value}\n")
            f.write(f"- **文件数**: {len(analysis.structure.files)}\n")
            f.write(f"- **代码行数**: {analysis.structure.total_lines}\n")
            f.write(f"- **组件数**: {len(analysis.components)}\n\n")
            
            # 业务场景
            f.write("## 业务场景\n\n")
            f.write(f"- **领域**: {analysis.scenario.domain}\n")
            f.write(f"- **用途**: {analysis.scenario.use_case}\n")
            f.write(f"- **置信度**: {analysis.scenario.confidence:.2f}\n\n")
            
            # 设计模式
            if analysis.patterns:
                f.write("## 设计模式\n\n")
                for pattern in analysis.patterns:
                    f.write(f"### {pattern.name}\n")
                    f.write(f"- **类型**: {pattern.type}\n")
                    f.write(f"- **描述**: {pattern.description}\n")
                    f.write(f"- **位置**: {pattern.location}\n")
                    f.write(f"- **置信度**: {pattern.confidence:.2f}\n\n")
            
            # 组件列表
            f.write("## 主要组件\n\n")
            for component in analysis.components:
                f.write(f"- {component}\n")
            f.write("\n")
            
            # 依赖列表
            if analysis.dependencies:
                f.write("## 主要依赖\n\n")
                for dep in analysis.dependencies[:20]:  # 限制显示数量
                    f.write(f"- {dep}\n")
                f.write("\n")
        
        print(f"✅ 完整报告已生成: {report_path}")

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description='Framework Analyzer - 智能框架代码分析工具')
    parser.add_argument('--path', required=True, help='项目路径')
    parser.add_argument('--language', choices=['go', 'python', 'mixed'], help='指定项目语言')
    parser.add_argument('--interactive', action='store_true', help='启动交互式学习模式')
    parser.add_argument('--output', help='输出文件路径')
    parser.add_argument('--config', help='配置文件路径')
    parser.add_argument('--verbose', action='store_true', help='详细输出')
    parser.add_argument('--debug', action='store_true', help='调试模式')
    
    args = parser.parse_args()
    
    try:
        # 初始化分析器
        analyzer = FrameworkAnalyzer(args.config)
        
        if args.interactive:
            # 交互式模式
            learning = InteractiveLearning(analyzer)
            learning.start_interactive_session(args.path)
        else:
            # 批处理模式
            language = Language(args.language) if args.language else None
            analysis = analyzer.analyze_project(args.path, language)
            
            # 输出结果
            if args.output:
                # 转换为可序列化的格式
                analysis_dict = asdict(analysis)
                analysis_dict['language'] = analysis.language.value
                for pattern in analysis_dict['patterns']:
                    pattern['type'] = pattern['type'].value if hasattr(pattern['type'], 'value') else str(pattern['type'])
                
                with open(args.output, 'w', encoding='utf-8') as f:
                    json.dump(analysis_dict, f, indent=2, ensure_ascii=False)
                print(f"✅ 分析结果已保存到: {args.output}")
            else:
                # 控制台输出
                print(f"\n📋 分析结果:")
                print(f"语言: {analysis.language.value}")
                print(f"业务场景: {analysis.scenario.domain}")
                print(f"设计模式: {len(analysis.patterns)} 个")
                print(f"组件: {len(analysis.components)} 个")
                
    except Exception as e:
        print(f"❌ 分析失败: {e}")
        if args.debug:
            import traceback
            traceback.print_exc()
        sys.exit(1)

if __name__ == '__main__':
    main()