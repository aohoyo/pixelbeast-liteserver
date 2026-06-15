/** 备份信息 */
export interface BackupInfo {
  name: string
  size: number
  modified: string
}

export interface BackupListResponse {
  backups: BackupInfo[]
  dir: string
}
