#!/usr/bin/env python3
"""
Visualizer - 框架可视化生成器

功能:
1. 生成Mermaid架构图
2. 生成依赖关系图
3. 生成数据流图
4. 生成组件交互图
"""

import os
import argparse
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from enum import Enum

class DiagramType(Enum):
    ARCHITECTURE = "architecture"
    DEPENDENCY = "dependency"
    DATAFLOW = "dataflow"
    INTERACTION = "interaction"

@dataclass
class Diagram:
    """图表数据结构"""
    type: DiagramType
    title: str
    content: str
    format: str = "mermaid"

class Visualizer:
    """可视化生成器"""
    
    def __init__(self):
        self.colors = {
            'application': '#E1F5FE',
            'service': '#F3E5F5', 
            'middleware': '#E8F5E8',
            'database': '#FFF3E0',
            'cache': '#FFEBEE',
            'storage': '#F1F8E9'
        }
    
    def generate_architecture_diagram(self, analysis) -> str:
        """生成架构图"""
        
        # 根据分析结果确定架构类型
        if analysis.scenario.domain == "AI Application":
            return self._generate_ai_architecture(analysis)
        elif analysis.scenario.domain == "Microservice":
            return self._generate_microservice_architecture(analysis)
        elif analysis.scenario.domain == "Web Application":
            return self._generate_web_architecture(analysis)
        else:
            return self._generate_generic_architecture(analysis)
    
    def _generate_ai_architecture(self, analysis) -> str:
        """生成AI应用架构图"""
        
        mermaid_content = """```mermaid
graph TB
    subgraph "AI Application Framework"
        subgraph "Application Layer"
            App[Application Core]
            Config[Configuration]
        end
        
        subgraph "Service Layer"
            API[REST API Server]
            Routes[API Routes]
        end
        
        subgraph "AI Processing Layer"
            ML[ML Models]
            Embed[Embedding Service]
            Search[Vector Search]
        end
        
        subgraph "Storage Layer"
"""
        
        # 根据中间件添加存储组件
        middleware = analysis.scenario.middleware
        if 'Vector Database' in middleware:
            mermaid_content += "            VectorDB[Vector Database<br/>Milvus/Weaviate]\n"
        if 'Knowledge Graph' in middleware:
            mermaid_content += "            KG[Knowledge Graph<br/>Weaviate]\n"
        if 'Database' in middleware:
            mermaid_content += "            DB[Relational Database<br/>PostgreSQL]\n"
        if 'Cache' in middleware:
            mermaid_content += "            Cache[Cache Layer<br/>Redis]\n"
        if 'Object Storage' in middleware:
            mermaid_content += "            Storage[Object Storage<br/>MinIO]\n"
        
        mermaid_content += """        end
        
        subgraph "Infrastructure Layer"
"""
        
        if 'Service Discovery' in middleware:
            mermaid_content += "            SD[Service Discovery<br/>etcd]\n"
        
        mermaid_content += """            Logger[Logging System]
            Monitor[Monitoring]
        end
    end
    
    %% Connections
    App --> API
    API --> Routes
    Routes --> ML
    ML --> Embed
    Embed --> Search
"""
        
        # 添加存储连接
        if 'Vector Database' in middleware:
            mermaid_content += "    Search --> VectorDB\n"
        if 'Database' in middleware:
            mermaid_content += "    App --> DB\n"
        if 'Cache' in middleware:
            mermaid_content += "    API --> Cache\n"
        if 'Object Storage' in middleware:
            mermaid_content += "    ML --> Storage\n"
        if 'Service Discovery' in middleware:
            mermaid_content += "    App --> SD\n"
        
        mermaid_content += """    
    %% Styling
    classDef appLayer fill:#E1F5FE
    classDef serviceLayer fill:#F3E5F5
    classDef aiLayer fill:#E8F5E8
    classDef storageLayer fill:#FFF3E0
    classDef infraLayer fill:#FFEBEE
    
    class App,Config appLayer
    class API,Routes serviceLayer
    class ML,Embed,Search aiLayer
"""
        
        if 'Vector Database' in middleware:
            mermaid_content += "    class VectorDB storageLayer\n"
        if 'Database' in middleware:
            mermaid_content += "    class DB storageLayer\n"
        if 'Cache' in middleware:
            mermaid_content += "    class Cache storageLayer\n"
        
        mermaid_content += "```"
        
        return self._save_diagram("ai_architecture", mermaid_content, analysis.structure.root_path)
    
    def _generate_microservice_architecture(self, analysis) -> str:
        """生成微服务架构图"""
        
        mermaid_content = """```mermaid
graph TB
    subgraph "Microservice Architecture"
        subgraph "API Gateway Layer"
            Gateway[API Gateway]
            LB[Load Balancer]
        end
        
        subgraph "Service Layer"
"""
        
        # 根据组件生成服务
        for i, component in enumerate(analysis.components[:5]):
            service_name = component.replace('_', ' ').title()
            mermaid_content += f"            Service{i+1}[{service_name} Service]\n"
        
        mermaid_content += """        end
        
        subgraph "Data Layer"
"""
        
        middleware = analysis.scenario.middleware
        if 'Database' in middleware:
            mermaid_content += "            DB[Database Cluster]\n"
        if 'Cache' in middleware:
            mermaid_content += "            Cache[Distributed Cache]\n"
        if 'Object Storage' in middleware:
            mermaid_content += "            Storage[Object Storage]\n"
        
        mermaid_content += """        end
        
        subgraph "Infrastructure Layer"
"""
        
        if 'Service Discovery' in middleware:
            mermaid_content += "            Registry[Service Registry]\n"
        
        mermaid_content += """            Config[Config Server]
            Monitor[Monitoring]
        end
    end
    
    %% Connections
    Gateway --> LB
"""
        
        # 连接服务
        for i in range(min(5, len(analysis.components))):
            mermaid_content += f"    LB --> Service{i+1}\n"
        
        # 连接数据层
        for i in range(min(5, len(analysis.components))):
            if 'Database' in middleware:
                mermaid_content += f"    Service{i+1} --> DB\n"
            if 'Cache' in middleware:
                mermaid_content += f"    Service{i+1} --> Cache\n"
        
        if 'Service Discovery' in middleware:
            for i in range(min(5, len(analysis.components))):
                mermaid_content += f"    Service{i+1} --> Registry\n"
        
        mermaid_content += "```"
        
        return self._save_diagram("microservice_architecture", mermaid_content, analysis.structure.root_path)
    
    def _generate_web_architecture(self, analysis) -> str:
        """生成Web应用架构图"""
        
        mermaid_content = """```mermaid
graph TB
    subgraph "Web Application Architecture"
        subgraph "Presentation Layer"
            Web[Web Interface]
            API[REST API]
        end
        
        subgraph "Business Layer"
            Controller[Controllers]
            Service[Business Services]
            Middleware[Middleware]
        end
        
        subgraph "Data Layer"
            Model[Data Models]
"""
        
        middleware = analysis.scenario.middleware
        if 'Database' in middleware:
            mermaid_content += "            DB[Database]\n"
        if 'Cache' in middleware:
            mermaid_content += "            Cache[Cache]\n"
        
        mermaid_content += """        end
    end
    
    %% Connections
    Web --> Controller
    API --> Controller
    Controller --> Service
    Service --> Middleware
    Middleware --> Model
"""
        
        if 'Database' in middleware:
            mermaid_content += "    Model --> DB\n"
        if 'Cache' in middleware:
            mermaid_content += "    Service --> Cache\n"
        
        mermaid_content += "```"
        
        return self._save_diagram("web_architecture", mermaid_content, analysis.structure.root_path)
    
    def _generate_generic_architecture(self, analysis) -> str:
        """生成通用架构图"""
        
        mermaid_content = f"""```mermaid
graph TB
    subgraph "{Path(analysis.structure.root_path).name} Architecture"
        subgraph "Application Layer"
            App[Main Application]
"""
        
        # 添加主要组件
        for i, component in enumerate(analysis.components[:3]):
            comp_name = component.replace('_', ' ').title()
            mermaid_content += f"            Comp{i+1}[{comp_name}]\n"
        
        mermaid_content += """        end
        
        subgraph "Infrastructure Layer"
"""
        
        # 添加中间件
        middleware = analysis.scenario.middleware
        if middleware:
            for mw in middleware[:4]:
                mw_id = mw.replace(' ', '').replace('-', '')
                mermaid_content += f"            {mw_id}[{mw}]\n"
        
        mermaid_content += """        end
    end
    
    %% Connections
    App --> Comp1
"""
        
        # 添加连接
        for i in range(min(3, len(analysis.components))):
            if middleware:
                for mw in middleware[:2]:
                    mw_id = mw.replace(' ', '').replace('-', '')
                    mermaid_content += f"    Comp{i+1} --> {mw_id}\n"
        
        mermaid_content += "```"
        
        return self._save_diagram("generic_architecture", mermaid_content, analysis.structure.root_path)
    
    def generate_dependency_graph(self, analysis) -> str:
        """生成依赖关系图"""
        
        mermaid_content = """```mermaid
graph LR
    subgraph "External Dependencies"
"""
        
        # 添加外部依赖
        external_deps = []
        for dep in analysis.dependencies[:10]:
            if '/' in dep:  # 外部包
                dep_name = dep.split('/')[-1]
                dep_id = dep_name.replace('-', '').replace('.', '')
                external_deps.append((dep_id, dep_name))
                mermaid_content += f"        {dep_id}[{dep_name}]\n"
        
        mermaid_content += """    end
    
    subgraph "Internal Components"
"""
        
        # 添加内部组件
        for component in analysis.components[:8]:
            comp_id = component.replace('-', '').replace('_', '')
            comp_name = component.replace('_', ' ').title()
            mermaid_content += f"        {comp_id}[{comp_name}]\n"
        
        mermaid_content += """    end
    
    %% Dependencies
"""
        
        # 添加依赖关系
        for component in analysis.components[:5]:
            comp_id = component.replace('-', '').replace('_', '')
            for dep_id, _ in external_deps[:3]:
                mermaid_content += f"    {comp_id} --> {dep_id}\n"
        
        mermaid_content += "```"
        
        return self._save_diagram("dependency_graph", mermaid_content, analysis.structure.root_path)
    
    def generate_dataflow_diagram(self, analysis) -> str:
        """生成数据流图"""
        
        mermaid_content = """```mermaid
flowchart LR
    subgraph "Data Input"
        Input[Data Input]
        Validation[Data Validation]
    end
    
    subgraph "Processing"
        Process[Data Processing]
        Transform[Data Transform]
    end
    
    subgraph "Storage"
"""
        
        middleware = analysis.scenario.middleware
        if 'Database' in middleware:
            mermaid_content += "        DB[(Database)]\n"
        if 'Cache' in middleware:
            mermaid_content += "        Cache[(Cache)]\n"
        if 'Vector Database' in middleware:
            mermaid_content += "        VectorDB[(Vector DB)]\n"
        if 'Object Storage' in middleware:
            mermaid_content += "        Storage[(Object Storage)]\n"
        
        mermaid_content += """    end
    
    subgraph "Output"
        API[API Response]
        UI[User Interface]
    end
    
    %% Data Flow
    Input --> Validation
    Validation --> Process
    Process --> Transform
"""
        
        if 'Database' in middleware:
            mermaid_content += "    Transform --> DB\n"
            mermaid_content += "    DB --> API\n"
        if 'Cache' in middleware:
            mermaid_content += "    Process --> Cache\n"
            mermaid_content += "    Cache --> API\n"
        if 'Vector Database' in middleware:
            mermaid_content += "    Transform --> VectorDB\n"
            mermaid_content += "    VectorDB --> API\n"
        
        mermaid_content += """    API --> UI
    
    %% Styling
    classDef input fill:#E3F2FD
    classDef process fill:#F1F8E9
    classDef storage fill:#FFF3E0
    classDef output fill:#FCE4EC
    
    class Input,Validation input
    class Process,Transform process
"""
        
        if 'Database' in middleware:
            mermaid_content += "    class DB storage\n"
        if 'Cache' in middleware:
            mermaid_content += "    class Cache storage\n"
        
        mermaid_content += "    class API,UI output\n```"
        
        return self._save_diagram("dataflow_diagram", mermaid_content, analysis.structure.root_path)
    
    def generate_interaction_diagram(self, analysis) -> str:
        """生成组件交互时序图"""
        
        mermaid_content = """```mermaid
sequenceDiagram
    participant Client
    participant API
"""
        
        # 添加主要组件作为参与者
        for component in analysis.components[:4]:
            comp_name = component.replace('_', ' ').title()
            mermaid_content += f"    participant {component} as {comp_name}\n"
        
        middleware = analysis.scenario.middleware
        if 'Database' in middleware:
            mermaid_content += "    participant DB as Database\n"
        if 'Cache' in middleware:
            mermaid_content += "    participant Cache\n"
        
        mermaid_content += """
    %% Interaction Flow
    Client->>API: Request
    API->>API: Validate Request
"""
        
        # 添加组件交互
        if analysis.components:
            first_comp = analysis.components[0]
            mermaid_content += f"    API->>{first_comp}: Process Request\n"
            
            if 'Cache' in middleware:
                mermaid_content += f"    {first_comp}->>Cache: Check Cache\n"
                mermaid_content += f"    Cache-->>{first_comp}: Cache Result\n"
            
            if 'Database' in middleware:
                mermaid_content += f"    {first_comp}->>DB: Query Data\n"
                mermaid_content += f"    DB-->>{first_comp}: Return Data\n"
            
            mermaid_content += f"    {first_comp}-->>API: Response Data\n"
        
        mermaid_content += """    API-->>Client: Final Response
```"""
        
        return self._save_diagram("interaction_diagram", mermaid_content, analysis.structure.root_path)
    
    def _save_diagram(self, name: str, content: str, root_path: str) -> str:
        """保存图表到文件"""
        
        # 创建diagrams目录
        diagrams_dir = Path(root_path) / "diagrams"
        diagrams_dir.mkdir(exist_ok=True)
        
        # 保存Mermaid文件
        mermaid_file = diagrams_dir / f"{name}.md"
        with open(mermaid_file, 'w', encoding='utf-8') as f:
            f.write(f"# {name.replace('_', ' ').title()}\n\n")
            f.write(content)
        
        return str(mermaid_file)
    
    def generate_all_diagrams(self, analysis) -> Dict[str, str]:
        """生成所有图表"""
        
        diagrams = {}
        
        print("📊 正在生成架构图...")
        diagrams['architecture'] = self.generate_architecture_diagram(analysis)
        
        print("📊 正在生成依赖关系图...")
        diagrams['dependency'] = self.generate_dependency_graph(analysis)
        
        print("📊 正在生成数据流图...")
        diagrams['dataflow'] = self.generate_dataflow_diagram(analysis)
        
        print("📊 正在生成交互图...")
        diagrams['interaction'] = self.generate_interaction_diagram(analysis)
        
        return diagrams
    
    def create_diagram_index(self, diagrams: Dict[str, str], root_path: str):
        """创建图表索引页面"""
        
        index_path = Path(root_path) / "diagrams" / "README.md"
        
        with open(index_path, 'w', encoding='utf-8') as f:
            f.write("# 框架架构图表\n\n")
            f.write("本目录包含了框架的各种架构图表，帮助理解系统设计。\n\n")
            
            f.write("## 图表列表\n\n")
            
            diagram_descriptions = {
                'architecture': '系统整体架构图，展示主要组件和层次结构',
                'dependency': '依赖关系图，展示组件间的依赖关系',
                'dataflow': '数据流图，展示数据在系统中的流转过程',
                'interaction': '交互时序图，展示组件间的交互流程'
            }
            
            for diagram_type, file_path in diagrams.items():
                file_name = Path(file_path).name
                description = diagram_descriptions.get(diagram_type, '系统图表')
                f.write(f"- [{file_name}](./{file_name}) - {description}\n")
            
            f.write("\n## 使用说明\n\n")
            f.write("这些图表使用Mermaid格式生成，可以在支持Mermaid的Markdown编辑器中查看，")
            f.write("如GitHub、GitLab、Typora等。\n\n")
            
            f.write("### 在线查看\n\n")
            f.write("如果您的编辑器不支持Mermaid，可以使用以下在线工具：\n")
            f.write("- [Mermaid Live Editor](https://mermaid.live/)\n")
            f.write("- [GitHub Mermaid Support](https://github.blog/2022-02-14-include-diagrams-markdown-files-mermaid/)\n")

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description='Framework Visualizer - 框架可视化生成器')
    parser.add_argument('--path', required=True, help='项目路径')
    parser.add_argument('--type', choices=['architecture', 'dependency', 'dataflow', 'interaction', 'all'], 
                       default='all', help='图表类型')
    parser.add_argument('--output', help='输出目录')
    parser.add_argument('--format', choices=['mermaid', 'plantuml'], default='mermaid', help='图表格式')
    
    args = parser.parse_args()
    
    try:
        # 这里需要从analyzer.py导入分析结果
        # 为了演示，我们创建一个简单的分析结果
        from analyzer import FrameworkAnalyzer
        
        analyzer = FrameworkAnalyzer()
        analysis = analyzer.analyze_project(args.path)
        
        visualizer = Visualizer()
        
        if args.type == 'all':
            diagrams = visualizer.generate_all_diagrams(analysis)
            visualizer.create_diagram_index(diagrams, args.output or args.path)
            print(f"✅ 所有图表已生成到: {args.output or args.path}/diagrams/")
        else:
            if args.type == 'architecture':
                result = visualizer.generate_architecture_diagram(analysis)
            elif args.type == 'dependency':
                result = visualizer.generate_dependency_graph(analysis)
            elif args.type == 'dataflow':
                result = visualizer.generate_dataflow_diagram(analysis)
            elif args.type == 'interaction':
                result = visualizer.generate_interaction_diagram(analysis)
            
            print(f"✅ {args.type}图表已生成: {result}")
            
    except Exception as e:
        print(f"❌ 生成图表失败: {e}")
        import traceback
        traceback.print_exc()

if __name__ == '__main__':
    main()