package projection

import "gorm.io/gorm"

// Offset tracks the last processed Kafka offset per consumer group+topic+partition.
// Stored in the same DB as projections for atomic updates.
type Offset struct {
	ID           uint   `gorm:"primaryKey"`
	ConsumerGroup string `gorm:"uniqueIndex:idx_group_topic_partition;type:varchar(255)"`
	Topic        string `gorm:"uniqueIndex:idx_group_topic_partition;type:varchar(255)"`
	Partition    int32  `gorm:"uniqueIndex:idx_group_topic_partition"`
	Offset       int64
}

// SaveOffset upserts the offset within an existing GORM transaction.
func SaveOffset(tx *gorm.DB, consumerGroup, topic string, partition int32, offset int64) error {
	var existing Offset
	result := tx.Where("consumer_group = ? AND topic = ? AND partition = ?", consumerGroup, topic, partition).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		return tx.Create(&Offset{
			ConsumerGroup: consumerGroup,
			Topic:         topic,
			Partition:     partition,
			Offset:        offset,
		}).Error
	}

	if result.Error != nil {
		return result.Error
	}

	// Only advance forward
	if offset > existing.Offset {
		return tx.Model(&existing).Update("offset", offset).Error
	}
	return nil
}
