/** 文件条目（GET /api/files） */
export interface FileEntry {
  name: string
  is_dir: boolean
  size: number
  modified: string
}

export interface FileListResponse {
  path: string
  program_dir: string
  files: FileEntry[]
}

export interface QuickDir {
  path?: string
  name?: string
  icon?: string
  section?: string
  isDefault?: boolean
  pinned?: boolean
  editable?: boolean
}

export interface QuickDirsResponse {
  dirs: QuickDir[]
  program_dir: string
}

/** 文件读取（编辑器） */
export interface FileReadResponse {
  content: string
  size: number
  modified: string
  type: string
  path: string
  name: string
}

/** 回收站条目 */
export interface TrashItem {
  id: string
  original_path: string
  original_name: string
  deleted_at: string
  is_dir: boolean
  size: number
}

export interface TrashListResponse {
  items: TrashItem[]
  total: number
}

/** 分享链接 */
export interface ShareLink {
  token: string
  fileName: string
  fileSize: number
  expiresAt: string
  createdAt: string
  downloadCount: number
}
