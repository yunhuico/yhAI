# 云辉大模型管理平台

云辉大模型管理平台是一个开源平台，专为管理大语言模型（LLM）资产而设计，提供了高效管理 LLM 及其资产（如数据集、应用空间、代码等）的方式。用户可以通过网页界面、git 命令行， LLM 资产进
行上传、下载、存储、校验和分发。

云辉大模型管理平台为大模型提供了友好的管理平台，并支持本地化部署，确保安全、离线运行。

## 核心功能

- **统一管理大模型资产**
- **灵活兼容的开发生态系统**
- **大模型能力扩展**
- **应用空间与资产管理助手（Copilot）**
- **支持私有化部署**
- **一站式数据处理与智能标注系统**
- **高可用与灾难恢复设计**
- **RESTful API**

## 功能描述

云辉大模型平台整合了 Git 服务器、Git LFS（大文件存储）协议以及对象存储服务（OSS），为数据存储提供了坚实的基础，确保灵活的基础设施访问，并为开发工具提供全面支持。

该平台采用服务导向架构，通过云辉服务器提供后端服务，同时通过 Web 服务提供友好的管理界面。普通用户可以通过 Docker Compose 或 Kubernetes Helm Chart 快速部署服务，实现企业级资产管理。>此外，具备开发能力的用户可以利用云辉服务器进行二次开发，将管理功能与外部系统集成，或定制更多高级功能。

云辉大模型平台支持 Parquet 数据文件格式的预览，从而便捷地管理本地数据集。平台提供直观的 Web 界面和符合企业组织结构的权限控制，用户可通过 Web UI 实现版本控制、在线浏览及下载，设置数>据集和模型文件的访问权限，从而确保数据隔离和安全性。此外，用户还能够围绕模型和数据集开展主题讨论。

我们的研发团队专注于 AI 与 DevOps 的结合，旨在通过云辉大模型平台解决大模型开发过程中的痛点。我们鼓励用户积极提供优质的开发和运维文档，共同完善平台，推动大模型资产的丰富与高效发展。
~
~
