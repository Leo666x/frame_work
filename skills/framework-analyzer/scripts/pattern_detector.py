#!/usr/bin/env python3
"""
Pattern Detector - 设计模式检测器

功能:
1. 检测Go语言设计模式
2. 检测Python设计模式
3. 检测架构模式
4. 生成模式分析报告
"""

import os
import re
import ast
import argparse
from pathlib import Path
from typing import Dict, List, Optional, Set, Tuple
from dataclasses import dataclass
from enum import Enum
import json

class PatternType(Enum):
    CREATIONAL = "creational"
    STRUCTURAL = "structural"
    BEHAVIORAL = "behavioral"
    ARCHITECTURAL = "architectural"
    CONCURRENCY = "concurrency"

@dataclass
class PatternMatch:
    """模式匹配结果"""
    name: str
    type: PatternType
    description: str
    file_path: str
    line_number: int
    confidence: float
    code_snippet: str
    indicators: List[str]

@dataclass
class PatternAnalysis:
    """模式分析结果"""
    total_patterns: int
    patterns_by_type: Dict[PatternType, int]
    matches: List[PatternMatch]
    recommendations: List[str]

class GoPatternDetector:
    """Go语言设计模式检测器"""
    
    def __init__(self):
        self.patterns = {
            # 创建型模式
            'factory': {
                'type': PatternType.CREATIONAL,
                'indicators': [
                    r'func\s+New\w+\s*\(',
                    r'func\s+Create\w+\s*\(',
                    r'func\s+Make\w+\s*\(',
                    r'return\s+&\w+\{'
                ],
                'description': '工厂模式 - 用于创建对象的接口'
            },
            'builder': {
                'type': PatternType.CREATIONAL,
                'indicators': [
                    r'type\s+\w+Builder\s+struct',
                    r'func\s+\(\w+\s+\*\w+Builder\)\s+\w+\(',
                    r'func\s+\(\w+\s+\*\w+Builder\)\s+Build\(',
                    r'\.With\w+\('
                ],
                'description': '建造者模式 - 逐步构建复杂对象'
            },
            'singleton': {
                'type': PatternType.CREATIONAL,
                'indicators': [
                    r'sync\.Once',
                    r'var\s+instance\s+\*\w+',
                    r'func\s+GetInstance\(',
                    r'once\.Do\('
                ],
                'description': '单例模式 - 确保类只有一个实例'
            },
            
            # 结构型模式
            'adapter': {
                'type': PatternType.STRUCTURAL,
                'indicators': [
                    r'type\s+\w+Adapter\s+struct',
                    r'func\s+\(\w+\s+\*\w+Adapter\)',
                    r'// Adapter pattern'
                ],
                'description': '适配器模式 - 使不兼容的接口能够协同工作'
            },
            'decorator': {
                'type': PatternType.STRUCTURAL,
                'indicators': [
                    r'type\s+\w+Decorator\s+struct',
                    r'func\s+\w+Middleware\(',
                    r'return\s+func\(',
                    r'http\.HandlerFunc'
                ],
                'description': '装饰器模式 - 动态添加对象功能'
            },
            'facade': {
                'type': PatternType.STRUCTURAL,
                'indicators': [
                    r'type\s+\w+Facade\s+struct',
                    r'// Facade pattern',
                    r'func\s+New\w+Facade\('
                ],
                'description': '外观模式 - 为复杂子系统提供简单接口'
            },
            
            # 行为型模式
            'observer': {
                'type': PatternType.BEHAVIORAL,
                'indicators': [
                    r'type\s+\w+Observer\s+interface',
                    r'func\s+\(\w+\)\s+Notify\(',
                    r'func\s+\(\w+\)\s+Subscribe\(',
                    r'chan\s+\w+'
                ],
                'description': '观察者模式 - 对象状态改变时通知依赖对象'
            },
            'strategy': {
                'type': PatternType.BEHAVIORAL,
                'indicators': [
                    r'type\s+\w+Strategy\s+interface',
                    r'func\s+\(\w+\)\s+Execute\(',
                    r'switch\s+\w+\s*\{'
                ],
                'description': '策略模式 - 定义算法族并使其可互换'
            },
            'command': {
                'type': PatternType.BEHAVIORAL,
                'indicators': [
                    r'type\s+\w+Command\s+interface',
                    r'func\s+\(\w+\)\s+Execute\(',
                    r'type\s+\w+Invoker\s+struct'
                ],
                'description': '命令模式 - 将请求封装为对象'
            },
            
            # 并发模式
            'worker_pool': {
                'type': PatternType.CONCURRENCY,
                'indicators': [
                    r'make\(chan\s+\w+,\s*\d+\)',
                    r'go\s+func\(\)\s*\{',
                    r'for\s+\w+\s*:=\s*range\s+\w+Chan',
                    r'sync\.WaitGroup'
                ],
                'description': 'Worker Pool模式 - 使用固定数量的goroutine处理任务'
            },
            'pipeline': {
                'type': PatternType.CONCURRENCY,
                'indicators': [
                    r'<-chan\s+\w+',
                    r'chan<-\s+\w+',
                    r'go\s+\w+\(',
                    r'select\s*\{'
                ],
                'description': 'Pipeline模式 - 通过channel连接的处理阶段'
            },
            'fan_out_fan_in': {
                'type': PatternType.CONCURRENCY,
                'indicators': [
                    r'go\s+func\(\w+\s+<-chan',
                    r'merge\(',
                    r'for\s+i\s*:=\s*0;\s*i\s*<\s*\w+;\s*i\+\+',
                    r'go\s+\w+\(\w+,\s*\w+\)'
                ],
                'description': 'Fan-out/Fan-in模式 - 分发任务并收集结果'
            },
            
            # 架构模式
            'dependency_injection': {
                'type': PatternType.ARCHITECTURAL,
                'indicators': [
                    r'type\s+Option\s+func\(',
                    r'opts\s+\.\.\.\s*Option',
                    r'func\s+With\w+\(',
                    r'newOptions\('
                ],
                'description': '依赖注入模式 - 通过外部注入依赖'
            },
            'repository': {
                'type': PatternType.ARCHITECTURAL,
                'indicators': [
                    r'type\s+\w+Repository\s+interface',
                    r'func\s+\(\w+\)\s+Save\(',
                    r'func\s+\(\w+\)\s+FindBy\w+\(',
                    r'func\s+\(\w+\)\s+Delete\('
                ],
                'description': '仓储模式 - 封装数据访问逻辑'
            },
            'mvc': {
                'type': PatternType.ARCHITECTURAL,
                'indicators': [
                    r'type\s+\w+Controller\s+struct',
                    r'type\s+\w+Model\s+struct',
                    r'type\s+\w+View\s+struct',
                    r'func\s+\(\w+\)\s+Handle\w+\('
                ],
                'description': 'MVC模式 - Model-View-Controller架构'
            }
        }
    
    def detect_patterns_in_file(self, file_path: Path) -> List[PatternMatch]:
        """检测单个文件中的模式"""
        matches = []
        
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                lines = content.split('\n')
            
            for pattern_name, pattern_info in self.patterns.items():
                pattern_matches = self._find_pattern_matches(
                    pattern_name, pattern_info, content, lines, str(file_path)
                )
                matches.extend(pattern_matches)
                
        except Exception as e:
            print(f"⚠️  检测文件 {file_path} 时出错: {e}")
        
        return matches
    
    def _find_pattern_matches(self, pattern_name: str, pattern_info: Dict, 
                            content: str, lines: List[str], file_path: str) -> List[PatternMatch]:
        """查找特定模式的匹配"""
        matches = []
        indicators = pattern_info['indicators']
        found_indicators = []
        
        # 检查每个指示器
        for indicator in indicators:
            regex_matches = re.finditer(indicator, content, re.MULTILINE)
            for match in regex_matches:
                # 计算行号
                line_num = content[:match.start()].count('\n') + 1
                found_indicators.append(indicator)
                
                # 获取代码片段
                start_line = max(0, line_num - 2)
                end_line = min(len(lines), line_num + 2)
                code_snippet = '\n'.join(lines[start_line:end_line])
                
                # 计算置信度
                confidence = len(set(found_indicators)) / len(indicators)
                
                if confidence >= 0.3:  # 至少匹配30%的指示器
                    matches.append(PatternMatch(
                        name=pattern_name.replace('_', ' ').title(),
                        type=pattern_info['type'],
                        description=pattern_info['description'],
                        file_path=file_path,
                        line_number=line_num,
                        confidence=confidence,
                        code_snippet=code_snippet,
                        indicators=list(set(found_indicators))
                    ))
        
        return matches

class PythonPatternDetector:
    """Python设计模式检测器"""
    
    def __init__(self):
        self.patterns = {
            # 创建型模式
            'factory': {
                'type': PatternType.CREATIONAL,
                'indicators': [
                    r'def\s+create_\w+\(',
                    r'class\s+\w+Factory',
                    r'@staticmethod',
                    r'return\s+\w+\('
                ],
                'description': '工厂模式 - 创建对象的接口'
            },
            'singleton': {
                'type': PatternType.CREATIONAL,
                'indicators': [
                    r'__new__\(',
                    r'_instance\s*=\s*None',
                    r'if\s+not\s+hasattr\(',
                    r'@singleton'
                ],
                'description': '单例模式 - 确保类只有一个实例'
            },
            
            # 结构型模式
            'decorator': {
                'type': PatternType.STRUCTURAL,
                'indicators': [
                    r'@\w+',
                    r'def\s+\w+\(func\)',
                    r'functools\.wraps',
                    r'return\s+wrapper'
                ],
                'description': '装饰器模式 - 动态添加功能'
            },
            'adapter': {
                'type': PatternType.STRUCTURAL,
                'indicators': [
                    r'class\s+\w+Adapter',
                    r'def\s+__init__\(self,\s*adaptee\)',
                    r'self\._adaptee'
                ],
                'description': '适配器模式 - 接口适配'
            },
            
            # 行为型模式
            'observer': {
                'type': PatternType.BEHAVIORAL,
                'indicators': [
                    r'class\s+\w+Observer',
                    r'def\s+notify\(',
                    r'def\s+subscribe\(',
                    r'self\._observers'
                ],
                'description': '观察者模式 - 状态变化通知'
            },
            'strategy': {
                'type': PatternType.BEHAVIORAL,
                'indicators': [
                    r'class\s+\w+Strategy',
                    r'def\s+execute\(',
                    r'abc\.ABC',
                    r'@abstractmethod'
                ],
                'description': '策略模式 - 算法族可互换'
            },
            
            # 架构模式
            'mvc': {
                'type': PatternType.ARCHITECTURAL,
                'indicators': [
                    r'class\s+\w+Model',
                    r'class\s+\w+View',
                    r'class\s+\w+Controller',
                    r'from\s+django',
                    r'from\s+flask'
                ],
                'description': 'MVC模式 - Model-View-Controller'
            },
            'repository': {
                'type': PatternType.ARCHITECTURAL,
                'indicators': [
                    r'class\s+\w+Repository',
                    r'def\s+save\(',
                    r'def\s+find_by_\w+\(',
                    r'def\s+delete\('
                ],
                'description': '仓储模式 - 数据访问封装'
            }
        }
    
    def detect_patterns_in_file(self, file_path: Path) -> List[PatternMatch]:
        """检测Python文件中的模式"""
        matches = []
        
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                lines = content.split('\n')
            
            # 尝试解析AST
            try:
                tree = ast.parse(content)
                ast_matches = self._analyze_ast(tree, str(file_path), lines)
                matches.extend(ast_matches)
            except SyntaxError:
                pass  # 如果AST解析失败，继续使用正则表达式
            
            # 使用正则表达式检测
            for pattern_name, pattern_info in self.patterns.items():
                pattern_matches = self._find_pattern_matches(
                    pattern_name, pattern_info, content, lines, str(file_path)
                )
                matches.extend(pattern_matches)
                
        except Exception as e:
            print(f"⚠️  检测Python文件 {file_path} 时出错: {e}")
        
        return matches
    
    def _analyze_ast(self, tree: ast.AST, file_path: str, lines: List[str]) -> List[PatternMatch]:
        """通过AST分析检测模式"""
        matches = []
        
        class PatternVisitor(ast.NodeVisitor):
            def __init__(self):
                self.decorators_found = []
                self.classes_found = []
                self.methods_found = []
            
            def visit_ClassDef(self, node):
                self.classes_found.append(node.name)
                
                # 检测装饰器
                for decorator in node.decorator_list:
                    if isinstance(decorator, ast.Name):
                        self.decorators_found.append(decorator.id)
                
                self.generic_visit(node)
            
            def visit_FunctionDef(self, node):
                self.methods_found.append(node.name)
                
                # 检测装饰器
                for decorator in node.decorator_list:
                    if isinstance(decorator, ast.Name):
                        self.decorators_found.append(decorator.id)
                
                self.generic_visit(node)
        
        visitor = PatternVisitor()
        visitor.visit(tree)
        
        # 基于AST信息检测模式
        if visitor.decorators_found:
            matches.append(PatternMatch(
                name="Decorator Pattern",
                type=PatternType.STRUCTURAL,
                description="Python装饰器模式",
                file_path=file_path,
                line_number=1,
                confidence=0.9,
                code_snippet="# Decorators found: " + ", ".join(visitor.decorators_found),
                indicators=visitor.decorators_found
            ))
        
        # 检测MVC模式
        mvc_indicators = []
        for class_name in visitor.classes_found:
            if 'Model' in class_name:
                mvc_indicators.append('Model')
            elif 'View' in class_name:
                mvc_indicators.append('View')
            elif 'Controller' in class_name:
                mvc_indicators.append('Controller')
        
        if len(mvc_indicators) >= 2:
            matches.append(PatternMatch(
                name="MVC Pattern",
                type=PatternType.ARCHITECTURAL,
                description="Model-View-Controller架构模式",
                file_path=file_path,
                line_number=1,
                confidence=len(mvc_indicators) / 3.0,
                code_snippet="# MVC components found",
                indicators=mvc_indicators
            ))
        
        return matches
    
    def _find_pattern_matches(self, pattern_name: str, pattern_info: Dict,
                            content: str, lines: List[str], file_path: str) -> List[PatternMatch]:
        """查找特定模式的匹配"""
        matches = []
        indicators = pattern_info['indicators']
        found_indicators = []
        
        for indicator in indicators:
            regex_matches = re.finditer(indicator, content, re.MULTILINE)
            for match in regex_matches:
                line_num = content[:match.start()].count('\n') + 1
                found_indicators.append(indicator)
                
                start_line = max(0, line_num - 2)
                end_line = min(len(lines), line_num + 2)
                code_snippet = '\n'.join(lines[start_line:end_line])
                
                confidence = len(set(found_indicators)) / len(indicators)
                
                if confidence >= 0.3:
                    matches.append(PatternMatch(
                        name=pattern_name.replace('_', ' ').title(),
                        type=pattern_info['type'],
                        description=pattern_info['description'],
                        file_path=file_path,
                        line_number=line_num,
                        confidence=confidence,
                        code_snippet=code_snippet,
                        indicators=list(set(found_indicators))
                    ))
        
        return matches

class PatternDetector:
    """主模式检测器"""
    
    def __init__(self):
        self.go_detector = GoPatternDetector()
        self.python_detector = PythonPatternDetector()
    
    def analyze_project(self, project_path: str, patterns: Optional[List[str]] = None) -> PatternAnalysis:
        """分析项目中的设计模式"""
        print(f"🔍 正在分析项目模式: {project_path}")
        
        project_root = Path(project_path)
        if not project_root.exists():
            raise FileNotFoundError(f"项目路径不存在: {project_path}")
        
        all_matches = []
        
        # 扫描Go文件
        go_files = list(project_root.rglob('*.go'))
        if go_files:
            print(f"📁 发现 {len(go_files)} 个Go文件")
            for go_file in go_files:
                if self._should_skip_file(go_file):
                    continue
                matches = self.go_detector.detect_patterns_in_file(go_file)
                all_matches.extend(matches)
        
        # 扫描Python文件
        py_files = list(project_root.rglob('*.py'))
        if py_files:
            print(f"📁 发现 {len(py_files)} 个Python文件")
            for py_file in py_files:
                if self._should_skip_file(py_file):
                    continue
                matches = self.python_detector.detect_patterns_in_file(py_file)
                all_matches.extend(matches)
        
        # 过滤指定的模式
        if patterns:
            pattern_names = [p.lower().replace(' ', '_') for p in patterns]
            all_matches = [m for m in all_matches if m.name.lower().replace(' ', '_') in pattern_names]
        
        # 去重和排序
        unique_matches = self._deduplicate_matches(all_matches)
        unique_matches.sort(key=lambda x: x.confidence, reverse=True)
        
        # 统计分析
        patterns_by_type = {}
        for match in unique_matches:
            if match.type not in patterns_by_type:
                patterns_by_type[match.type] = 0
            patterns_by_type[match.type] += 1
        
        # 生成建议
        recommendations = self._generate_recommendations(unique_matches, patterns_by_type)
        
        analysis = PatternAnalysis(
            total_patterns=len(unique_matches),
            patterns_by_type=patterns_by_type,
            matches=unique_matches,
            recommendations=recommendations
        )
        
        print(f"✅ 模式分析完成: 发现 {len(unique_matches)} 个模式")
        return analysis
    
    def _should_skip_file(self, file_path: Path) -> bool:
        """判断是否应该跳过文件"""
        skip_dirs = {'vendor', 'node_modules', '.git', '__pycache__', 'venv', '.venv'}
        skip_files = {'_test.go', 'test_*.py'}
        
        # 检查目录
        if any(skip_dir in file_path.parts for skip_dir in skip_dirs):
            return True
        
        # 检查文件名
        if any(file_path.name.endswith(skip_file.replace('*', '')) or 
               file_path.name.startswith(skip_file.replace('*', '')) 
               for skip_file in skip_files):
            return True
        
        return False
    
    def _deduplicate_matches(self, matches: List[PatternMatch]) -> List[PatternMatch]:
        """去除重复的模式匹配"""
        seen = set()
        unique_matches = []
        
        for match in matches:
            # 创建唯一标识符
            key = (match.name, match.file_path, match.line_number)
            if key not in seen:
                seen.add(key)
                unique_matches.append(match)
        
        return unique_matches
    
    def _generate_recommendations(self, matches: List[PatternMatch], 
                                patterns_by_type: Dict[PatternType, int]) -> List[str]:
        """生成模式使用建议"""
        recommendations = []
        
        # 基于发现的模式类型给出建议
        if PatternType.CREATIONAL in patterns_by_type:
            recommendations.append("✅ 发现创建型模式，有助于对象创建的灵活性")
        
        if PatternType.STRUCTURAL in patterns_by_type:
            recommendations.append("✅ 发现结构型模式，有助于组件间的协作")
        
        if PatternType.BEHAVIORAL in patterns_by_type:
            recommendations.append("✅ 发现行为型模式，有助于算法和职责的分离")
        
        if PatternType.CONCURRENCY in patterns_by_type:
            recommendations.append("✅ 发现并发模式，有助于并发处理的优化")
        
        if PatternType.ARCHITECTURAL in patterns_by_type:
            recommendations.append("✅ 发现架构模式，有助于系统整体设计")
        
        # 基于模式数量给出建议
        total_patterns = len(matches)
        if total_patterns == 0:
            recommendations.append("💡 建议引入一些设计模式来提高代码质量")
        elif total_patterns < 5:
            recommendations.append("💡 可以考虑引入更多设计模式来提高代码的可维护性")
        elif total_patterns > 20:
            recommendations.append("⚠️  模式使用较多，注意避免过度设计")
        
        # 基于置信度给出建议
        high_confidence_patterns = [m for m in matches if m.confidence > 0.8]
        if len(high_confidence_patterns) / max(total_patterns, 1) > 0.7:
            recommendations.append("✅ 大部分模式实现质量较高")
        else:
            recommendations.append("💡 建议完善模式实现，提高代码规范性")
        
        return recommendations
    
    def generate_report(self, analysis: PatternAnalysis, output_path: Optional[str] = None) -> str:
        """生成模式分析报告"""
        
        report_lines = [
            "# 设计模式分析报告\n",
            f"## 概览\n",
            f"- **总模式数**: {analysis.total_patterns}\n",
            f"- **模式类型**: {len(analysis.patterns_by_type)}\n\n"
        ]
        
        # 按类型统计
        if analysis.patterns_by_type:
            report_lines.append("## 模式类型分布\n\n")
            for pattern_type, count in analysis.patterns_by_type.items():
                type_name = pattern_type.value.replace('_', ' ').title()
                report_lines.append(f"- **{type_name}**: {count} 个\n")
            report_lines.append("\n")
        
        # 详细模式列表
        if analysis.matches:
            report_lines.append("## 检测到的模式\n\n")
            
            current_type = None
            for match in analysis.matches:
                if match.type != current_type:
                    current_type = match.type
                    type_name = current_type.value.replace('_', ' ').title()
                    report_lines.append(f"### {type_name}模式\n\n")
                
                report_lines.extend([
                    f"#### {match.name}\n",
                    f"- **描述**: {match.description}\n",
                    f"- **文件**: {match.file_path}\n",
                    f"- **行号**: {match.line_number}\n",
                    f"- **置信度**: {match.confidence:.2f}\n",
                    f"- **指示器**: {', '.join(match.indicators)}\n\n",
                    "```\n",
                    match.code_snippet,
                    "\n```\n\n"
                ])
        
        # 建议
        if analysis.recommendations:
            report_lines.append("## 建议\n\n")
            for rec in analysis.recommendations:
                report_lines.append(f"- {rec}\n")
            report_lines.append("\n")
        
        report_content = ''.join(report_lines)
        
        # 保存报告
        if output_path:
            with open(output_path, 'w', encoding='utf-8') as f:
                f.write(report_content)
            print(f"📝 报告已保存到: {output_path}")
        
        return report_content

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description='Pattern Detector - 设计模式检测器')
    parser.add_argument('--path', required=True, help='项目路径')
    parser.add_argument('--patterns', help='指定要检测的模式，用逗号分隔 (如: factory,singleton,observer)')
    parser.add_argument('--output', help='输出报告文件路径')
    parser.add_argument('--format', choices=['markdown', 'json'], default='markdown', help='输出格式')
    parser.add_argument('--verbose', action='store_true', help='详细输出')
    
    args = parser.parse_args()
    
    try:
        detector = PatternDetector()
        
        # 解析指定的模式
        patterns = None
        if args.patterns:
            patterns = [p.strip() for p in args.patterns.split(',')]
            print(f"🎯 检测指定模式: {', '.join(patterns)}")
        
        # 分析项目
        analysis = detector.analyze_project(args.path, patterns)
        
        # 输出结果
        if args.format == 'json':
            # JSON格式输出
            result = {
                'total_patterns': analysis.total_patterns,
                'patterns_by_type': {k.value: v for k, v in analysis.patterns_by_type.items()},
                'matches': [
                    {
                        'name': m.name,
                        'type': m.type.value,
                        'description': m.description,
                        'file_path': m.file_path,
                        'line_number': m.line_number,
                        'confidence': m.confidence,
                        'indicators': m.indicators
                    }
                    for m in analysis.matches
                ],
                'recommendations': analysis.recommendations
            }
            
            if args.output:
                with open(args.output, 'w', encoding='utf-8') as f:
                    json.dump(result, f, indent=2, ensure_ascii=False)
            else:
                print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            # Markdown格式输出
            report = detector.generate_report(analysis, args.output)
            if not args.output:
                print(report)
        
        # 控制台摘要
        print(f"\n📊 分析摘要:")
        print(f"   总模式数: {analysis.total_patterns}")
        for pattern_type, count in analysis.patterns_by_type.items():
            type_name = pattern_type.value.replace('_', ' ').title()
            print(f"   {type_name}: {count}")
        
        if args.verbose and analysis.matches:
            print(f"\n🔍 详细结果:")
            for match in analysis.matches[:10]:  # 只显示前10个
                print(f"   - {match.name} ({match.confidence:.2f}) in {match.file_path}:{match.line_number}")
        
    except Exception as e:
        print(f"❌ 模式检测失败: {e}")
        import traceback
        traceback.print_exc()

if __name__ == '__main__':
    main()